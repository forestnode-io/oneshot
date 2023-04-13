package root

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/pkg/configuration"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
	"github.com/raphaelreyna/oneshot/v2/pkg/version"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func (r *rootCommand) init(cmd *cobra.Command, args []string) error {
	var ctx = cmd.Context()

	r.config.MergeFlags()
	if err := r.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	if err := r.config.Hydrate(); err != nil {
		return fmt.Errorf("failed to hydrate configuration: %w", err)
	}

	if r.config.Output.Quiet {
		output.Quiet(ctx)
	} else {
		output.SetFormat(ctx, r.config.Output.Format.Format)
		output.SetFormatOpts(ctx, r.config.Output.Format.Opts...)
	}
	if r.config.Output.NoColor {
		output.NoColor(ctx)
	}

	return nil
}

// runServer starts the actual oneshot http server.
// this should only be run after a subcommand since it relies on
// a subcommand to have set r.handler.
func (r *rootCommand) runServer(cmd *cobra.Command, args []string) error {
	var (
		ctx, cancel = context.WithCancel(cmd.Context())
		log         = zerolog.Ctx(ctx)
		err         error

		webRTCError error

		ntConf  = r.config.NATTraversal
		dsConf  = ntConf.DiscoveryServer
		p2pConf = ntConf.P2P
		baConf  = r.config.BasicAuth
	)
	defer cancel()

	defer func() {
		log.Debug().Msg("stopping events")
		events.Stop(ctx)

		log.Debug().Msg("waiting for output to finish")
		output.Wait(ctx)

		log.Debug().Msg("waiting for http server to close")
		r.wg.Wait()
		log.Debug().Msg("all network connections closed")
	}()

	if r.handler == nil {
		return nil
	}

	// handle port mapping ( this can take a while )
	externalAddr_UPnP, cancelPortMapping, err := r.handlePortMap(ctx)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to negotiate port mapping")

		return fmt.Errorf("failed to negotiate port mapping: %w", err)
	}
	defer cancelPortMapping()

	// connect to discovery server if one is provided
	var (
		discoveryServerURL   = dsConf.URL
		usingDiscoveryServer = discoveryServerURL != ""

		discoveryServerKey      = dsConf.Key
		discoveryServerInsecure = dsConf.Insecure
	)
	if discoveryServerURL != "" {
		dsConf := signallingserver.DiscoveryServerConfig{
			URL:      discoveryServerURL,
			Key:      discoveryServerKey,
			Insecure: discoveryServerInsecure,
			VersionInfo: messages.VersionInfo{
				Version:    version.Version,
				APIVersion: version.APIVersion,
			},
		}

		ctx, err = signallingserver.WithDiscoveryServer(ctx, dsConf)
		if err != nil {
			log.Error().Err(err).
				Msg("failed to connect to discovery server")

			return fmt.Errorf("failed to connect to discovery server: %w", err)
		}
	}

	baToken, err := r.configureServer()
	if err != nil {
		log.Error().Err(err).
			Msg("failed to configure server")

		return fmt.Errorf("failed to configure server: %w", err)
	}

	var (
		useWebRTC           = p2pConf.Enabled
		webRTCSignallingDir = p2pConf.DiscoveryDir
		webRTCSignallingURL = dsConf.URL
		webRTCOnly          = p2pConf.Only

		usingWebRTCWithDiscoveryServer = useWebRTC || webRTCOnly || webRTCSignallingURL != "" || usingDiscoveryServer
		usingWebRTC                    = usingWebRTCWithDiscoveryServer || webRTCSignallingDir != ""
		redirect                       = externalAddr_UPnP != ""

		discoveryServerAssignedAddress string
	)

	if usingDiscoveryServer && redirect && !usingWebRTCWithDiscoveryServer {
		// if we're using a discovery server but not webrtc and we're redirecting
		// then we need to still connect to the discovery server and send it the
		// redirect address
		ds := signallingserver.GetDiscoveryServer(ctx)
		if ds == nil {
			log.Error().Msg("discovery server is nil")

			return errors.New("discovery server is nil")
		}

		err = signallingserver.Send(ds, &messages.ServerArrivalRequest{
			Redirect:     externalAddr_UPnP,
			RedirectOnly: true,
		})
		if err != nil {
			log.Error().Err(err).
				Msg("failed to send server arrival request")

			return fmt.Errorf("failed to send server arrival request: %w", err)
		}
		resp, err := signallingserver.Receive[*messages.ServerArrivalResponse](ds)
		if err != nil {
			log.Error().Err(err).
				Msg("failed to receive server arrival response")

			return fmt.Errorf("failed to receive server arrival response: %w", err)
		}
		if resp.Error != "" {
			log.Error().
				Str("error", resp.Error).
				Msg("server arrival response error")

			return fmt.Errorf("server arrival response error: %s", resp.Error)
		}
		discoveryServerAssignedAddress = resp.AssignedURL
	} else if usingWebRTC {
		// if we're using webrtc then we need to listen for incoming connections
		// and send the discovery server our listening address
		var (
			externalAddrChan = make(chan string, 1)

			username = baConf.Username
			password = baConf.Password
			bam      *messages.BasicAuth
		)

		if username != "" || password != "" {
			bam = &messages.BasicAuth{}
			if username != "" {
				uHash := sha256.Sum256([]byte(username))
				bam.UsernameHash = uHash[:]
			}
			if password != "" {
				pHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					log.Error().Err(err).
						Msg("failed to hash password")

					return fmt.Errorf("failed to hash password: %w", err)
				}
				bam.PasswordHash = pHash
			}
		}
		go func() {
			if err := r.listenWebRTC(ctx, externalAddr_UPnP, baToken, externalAddrChan, bam); err != nil {
				log.Error().Err(err).
					Msg("failed to listen for WebRTC connections")

				webRTCError = err
				cancel()
			}
		}()

		log.Debug().Msg("waiting for WebRTC listening address")
		select {
		case <-ctx.Done():
		case discoveryServerAssignedAddress = <-externalAddrChan:
		}
		log.Debug().
			Str("address", discoveryServerAssignedAddress).
			Msg("got WebRTC listening address")
		if discoveryServerAssignedAddress == "" {
			return errors.New("unable to establish a connection with the discovery server")
		}
	}

	port := r.config.Server.Port

	userFacingAddr := discoveryServerAssignedAddress
	if userFacingAddr == "" {
		userFacingAddr = externalAddr_UPnP
	}
	if userFacingAddr == "" {
		// if we weren't given a listening addr,
		// try giving the ip address that can reach the
		// default gateway in the print out
		sourceIP, err := oneshotnet.GetSourceIP("", 80)
		if err == nil {
			userFacingAddr = fmt.Sprintf("%s://%s:%d", "http", sourceIP, port)
		} else {
			userFacingAddr = fmt.Sprintf("%s://%s:%d", "http", "localhost", port)
		}
	}
	listeningAddr := oneshotfmt.Address(r.config.Server.Host, port)
	err = r.listenAndServe(ctx, listeningAddr, userFacingAddr)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to listen for http connections")
	} else {
		err = webRTCError
	}
	return err
}

func (r *rootCommand) listenAndServe(ctx context.Context, listeningAddr, userFacingAddr string) error {
	var (
		webrtcOnly = r.config.NATTraversal.P2P.Only

		l   net.Listener
		err error
	)

	if !webrtcOnly {
		l, err = net.Listen("tcp", listeningAddr)
		if err != nil {
			return err
		}
		defer l.Close()
	}

	// if we are using nat traversal show the user the external address
	if r.config.Output.QRCode {
		output.WriteListeningOnQR(ctx, userFacingAddr)
	} else {
		output.WriteListeningOn(ctx, userFacingAddr)
	}

	return r.server.Serve(ctx, l)
}

var ErrTimeout = errors.New("timeout")

func unauthenticatedHandler(triggerLogin bool, statCode int, content []byte) http.HandlerFunc {
	if statCode == 0 {
		statCode = http.StatusUnauthorized
	}
	if content == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			if triggerLogin {
				w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
			}
			w.WriteHeader(statCode)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if triggerLogin {
			w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
		}
		w.WriteHeader(statCode)
		_, _ = w.Write(content)
	}
}

func corsOptionsFromConfig(config *configuration.CORS) *cors.Options {
	if config == nil {
		return &cors.Options{}
	}

	return &cors.Options{
		AllowedOrigins:       config.AllowedOrigins,
		AllowedHeaders:       config.AllowedHeaders,
		MaxAge:               config.MaxAge,
		AllowCredentials:     config.AllowCredentials,
		AllowPrivateNetwork:  config.AllowPrivateNetwork,
		OptionsSuccessStatus: config.SuccessStatus,
	}
}

func wrappedFlagUsages(flags *pflag.FlagSet) string {
	w, _, err := term.GetSize(0)
	if err != nil {
		w = 80
	}

	return flags.FlagUsagesWrapped(w)
}

func handleUsageErrors(outputUsage func(), next func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := next(cmd, args)
		if _, ok := err.(output.UsageError); ok {
			outputUsage()
		}
		return err
	}
}
