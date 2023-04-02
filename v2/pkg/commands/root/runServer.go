package root

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/bcrypt"
)

func (r *rootCommand) init(cmd *cobra.Command, _ []string) {
	var (
		ctx   = cmd.Context()
		flags = cmd.Flags()
	)

	if quiet, _ := flags.GetBool("quiet"); quiet {
		output.Quiet(ctx)
	} else {
		output.SetFormat(ctx, r.outFlag.Format)
		output.SetFormatOpts(ctx, r.outFlag.Opts...)
	}
	if noColor, _ := flags.GetBool("no-color"); noColor {
		output.NoColor(ctx)
	}
}

// runServer starts the actual oneshot http server.
// this should only be run after a subcommand since it relies on
// a subcommand to have set r.handler.
func (r *rootCommand) runServer(cmd *cobra.Command, args []string) error {
	var (
		ctx, cancel = context.WithCancel(cmd.Context())
		flags       = cmd.Flags()
		err         error

		webRTCError error
	)
	defer cancel()

	defer func() {
		log.Println("stopping events")
		events.Stop(ctx)

		log.Println("waiting for output to finish")
		output.Wait(ctx)

		log.Println("waiting for network connections to close")
		r.wg.Wait()
		log.Println("all network connections closed")
	}()

	if r.handler == nil {
		return nil
	}

	// handle port mapping ( this can take a while )
	externalAddr_UPnP, cancelPortMapping, err := r.handlePortMap(ctx, flags)
	if err != nil {
		return fmt.Errorf("failed to handle port mapping: %w", err)
	}
	defer cancelPortMapping()

	// connect to discovery server if one is provided
	var (
		discoveryServerURL, _ = flags.GetString("discovery-server-url")
		usingDiscoveryServer  = discoveryServerURL != ""

		discoveryServerKey, _      = flags.GetString("discovery-server-key")
		discoveryServerInsecure, _ = flags.GetBool("discovery-server-insecure")
	)
	if discoveryServerURL != "" {
		dsConf := signallingserver.DiscoveryServerConfig{
			URL:      discoveryServerURL,
			Key:      discoveryServerKey,
			Insecure: discoveryServerInsecure,
			VersionInfo: messages.VersionInfo{
				Version:    "0.1.0",
				APIVersion: "0.1.0",
			},
		}

		ctx, err = signallingserver.WithDiscoveryServer(ctx, dsConf)
		if err != nil {
			return fmt.Errorf("failed to create discovery server: %w", err)
		}
	}

	baToken, err := r.configureServer(flags)
	if err != nil {
		return fmt.Errorf("failed to configure server: %w", err)
	}

	var (
		useWebRTC, _           = flags.GetBool("p2p")
		webRTCSignallingDir, _ = flags.GetString("p2p-discovery-dir")
		webRTCSignallingURL, _ = flags.GetString("p2p-discovery-server-url")
		webRTCOnly, _          = flags.GetBool("p2p-only")

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
			return errors.New("discovery server is nil")
		}

		err = signallingserver.Send(ds, &messages.ServerArrivalRequest{
			Redirect:     externalAddr_UPnP,
			RedirectOnly: true,
		})
		if err != nil {
			return fmt.Errorf("failed to send server arrival request: %w", err)
		}
		resp, err := signallingserver.Receive[*messages.ServerArrivalResponse](ds)
		if err != nil {
			return fmt.Errorf("failed to receive server arrival response: %w", err)
		}
		if resp.Error != "" {
			return fmt.Errorf("server arrival response error: %s", resp.Error)
		}
		discoveryServerAssignedAddress = resp.AssignedURL
	} else if usingWebRTC {
		// if we're using webrtc then we need to listen for incoming connections
		// and send the discovery server our listening address
		var (
			externalAddrChan = make(chan string, 1)

			username, _ = flags.GetString("username")
			password, _ = flags.GetString("password")
			bam         *messages.BasicAuth
		)

		if username == "" || password == "" {
			bam = &messages.BasicAuth{}
			if username == "" {
				uHash := sha256.Sum256([]byte(username))
				bam.UsernameHash = uHash[:]
			}
			if password == "" {
				pHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return fmt.Errorf("failed to hash password: %w", err)
				}
				bam.PasswordHash = pHash
			}
		}
		go func() {

			if err := r.listenWebRTC(ctx, externalAddr_UPnP, baToken, externalAddrChan, bam); err != nil {
				webRTCError = err
				log.Printf("failed to listen for WebRTC connections: %v", err)
				cancel()
			}
		}()

		discoveryServerAssignedAddress = <-externalAddrChan
		if discoveryServerAssignedAddress == "" {
			return errors.New("listening address is empty")
		}
	}

	listeningAddr := discoveryServerAssignedAddress
	if listeningAddr == "" {
		listeningAddr = externalAddr_UPnP
	}
	if listeningAddr == "" {
		listeningAddr = "http://localhost:" + flags.Lookup("port").Value.String()
	}

	err = r.listenAndServe(ctx, listeningAddr, flags)
	if err != nil {
		log.Printf("failed to listen for http connections: %v", err)
	} else {
		err = webRTCError
	}
	return err
}

func (r *rootCommand) listenAndServe(ctx context.Context, listeningAddr string, flags *pflag.FlagSet) error {
	var (
		host, _       = flags.GetString("host")
		port, _       = flags.GetInt("port")
		webrtcOnly, _ = flags.GetBool("webrtc-only")

		l   net.Listener
		err error
	)

	if !webrtcOnly {
		l, err = net.Listen("tcp", oneshotfmt.Address(host, port))
		if err != nil {
			return err
		}
		defer l.Close()
	}

	// if we are using nat traversal show the user the external address
	if qr, _ := flags.GetBool("qr-code"); qr {
		output.WriteListeningOnQR(ctx, listeningAddr)
	} else {
		output.WriteListeningOn(ctx, listeningAddr)
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
