package root

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

	baMessage, err := r.configureServer(flags)
	if err != nil {
		return fmt.Errorf("failed to configure server: %w", err)
	}

	go func() {
		if err := r.listenWebRTC(ctx, baMessage); err != nil {
			webRTCError = err
			log.Printf("failed to listen for WebRTC connections: %v", err)
			cancel()
		}
	}()

	cancelPortMapping, err := r.handlePortMap(ctx, flags)
	if err != nil {
		return fmt.Errorf("failed to handle port mapping: %w", err)
	}
	defer cancelPortMapping()

	err = r.listenAndServe(ctx, flags)
	if err != nil {
		log.Printf("failed to listen for http connections: %v", err)
	} else {
		err = webRTCError
	}
	return err
}

func (r *rootCommand) listenAndServe(ctx context.Context, flags *pflag.FlagSet) error {
	var (
		host, _       = flags.GetString("host")
		port, _       = flags.GetString("port")
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

	if qr, _ := flags.GetBool("qr-code"); qr {
		if r.externalIP != nil {
			externalPort, _ := flags.GetInt("external-port")
			ep := strconv.Itoa(externalPort)
			output.WriteListeningOnQR(ctx, "http", r.externalIP.String(), ep)
		} else {
			if host == "" {
				hostIP, err := oneshotnet.GetSourceIP("", "")
				if err == nil {
					host = hostIP
				}
			}
			output.WriteListeningOnQR(ctx, "http", host, port)
		}
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
