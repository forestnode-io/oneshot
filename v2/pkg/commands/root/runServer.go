package root

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
	"github.com/oneshot-uno/oneshot/v2/pkg/events"
	oneshotnet "github.com/oneshot-uno/oneshot/v2/pkg/net"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver/headers"
	"github.com/oneshot-uno/oneshot/v2/pkg/output"
	oneshotfmt "github.com/oneshot-uno/oneshot/v2/pkg/output/fmt"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func (r *rootCommand) init(cmd *cobra.Command, args []string) error {
	var ctx = cmd.Context()

	err := viper.Unmarshal(r.config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := r.config.Validate(); err != nil {
		return output.UsageErrorF("invalid configuration: %w", err)
	}
	if err := r.config.Hydrate(); err != nil {
		return fmt.Errorf("failed to hydrate configuration: %w", err)
	}

	if r.config.Output.Quiet {
		output.Quiet(ctx)
	} else {
		var (
			format string
			opts   = []string{}
		)
		if r.config.Output.Format != "" {
			parts := strings.Split(r.config.Output.Format, "=")
			if len(parts) == 0 || 2 < len(parts) {
				return fmt.Errorf("invalid output format: %s", r.config.Output.Format)
			}
			format = parts[0]
			if format != "json" {
				return fmt.Errorf("invalid output format: %s", r.config.Output.Format)
			}

			if len(parts) == 2 {
				opts = strings.Split(parts[1], ",")
			}
			for _, opt := range opts {
				if opt != "compact" && opt != "include-file-contents" && opt != "exclude-file-contents" {
					return fmt.Errorf("invalid output format option: %s", opt)
				}
			}
		}

		output.SetFormat(ctx, format)
		output.SetFormatOpts(ctx, opts...)
	}
	if r.config.Output.NoColor {
		output.NoColor(ctx)
	}

	return nil
}

func (r *rootCommand) errorSuppressor(next func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := next(cmd, args)
		// if the context was cancelled, then its not an error
		if cmd.Context().Err() != nil {
			if !output.IsPrintable(err) {
				err = nil
			}
		}
		return err
	}
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
	)
	defer cancel()

	subCmdName := ""
	subCmd, _, _ := cmd.Find(args)
	if subCmd != nil {
		subCmdName = subCmd.Name()
	}

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

	// initialize connection to discovery server
	ctx, err = r.withDiscoveryServer(ctx, subCmdName)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to connect to discovery server")

		return fmt.Errorf("failed to connect to discovery server: %w", err)
	}

	baToken, err := r.configureServer()
	if err != nil {
		log.Error().Err(err).
			Msg("failed to configure server")

		return fmt.Errorf("failed to configure server: %w", err)
	}

	if r.config.NATTraversal.IsUsingWebRTC() {
		go func() {
			if err := r.listenWebRTC(ctx, externalAddr_UPnP, baToken); err != nil {
				log.Error().Err(err).
					Msg("failed to listen for WebRTC connections")

				webRTCError = err
				cancel()
			}
		}()
	} else if ds := signallingserver.GetDiscoveryServer(ctx); ds != nil {
		go func() {
			stream := ds.Stream()
			if _, err := stream.Recv(); err != nil {
				// check if the discovery server closed the connection by user request
				if errors.Is(err, io.EOF) {
					trailer := stream.Trailer()
					values := trailer.Get(headers.ClosedByUser)
					if len(values) > 0 {
						v := values[0]
						if v == "true" {
							log.Info().
								Msg("discovery server closed connection by user request")
							cancel()
						}
					}
				}
			}
		}()
	}

	port := r.config.Server.Port
	userFacingAddr := ""
	ds := signallingserver.GetDiscoveryServer(ctx)
	if ds != nil {
		userFacingAddr = ds.AssignedURL
	}

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
			return output.WrapPrintable(err)
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
