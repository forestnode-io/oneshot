package root

import (
	"context"
	"errors"
	"net"
	"net/http"

	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func (r *rootCommand) init(cmd *cobra.Command, args []string) {
	var (
		ctx   = cmd.Context()
		flags = cmd.Flags()
	)

	output.Init(ctx)
	if quiet, _ := flags.GetBool("quiet"); quiet {
		output.Quiet(ctx)
	} else {
		output.SetFormat(ctx, r.outFlag.format)
		output.SetFormatOpts(ctx, r.outFlag.opts...)
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
		ctx   = cmd.Context()
		flags = cmd.Flags()
	)

	if r.handler == nil {
		return nil
	}

	if err := r.configureServer(flags); err != nil {
		return err
	}

	return r.listenAndServe(ctx, flags)
}

func (r *rootCommand) configureServer(flags *pflag.FlagSet) error {
	var (
		timeout, _   = flags.GetDuration("timeout")
		allowBots, _ = flags.GetBool("allow-bots")
		// TODO(raphaelreyna): allow the user to set this somehow
		unauthenticatedHandler = http.HandlerFunc(nil)
	)

	uname, passwd, err := usernamePassword(flags)
	if err != nil {
		return err
	}

	tlsCert, tlsKey, err := tlsCertAndKey(flags)
	if err != nil {
		return err
	}

	goneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	})

	r.server = oneshothttp.NewServer(r.Context(), r.handler, goneHandler, []oneshothttp.Middleware{
		r.middleware.
			// TODO(raphaelreyna): add CORS middleware
			Chain(oneshothttp.BotsMiddleware(allowBots)).
			Chain(oneshothttp.BasicAuthMiddleware(unauthenticatedHandler, uname, passwd)),
	}...)
	r.server.TLSCert = tlsCert
	r.server.TLSKey = tlsKey
	r.server.Timeout = timeout

	return nil
}

func (r *rootCommand) listenAndServe(ctx context.Context, flags *pflag.FlagSet) error {
	var (
		host, _ = flags.GetString("host")
		port, _ = flags.GetString("port")
	)
	l, err := net.Listen("tcp", oneshotfmt.Address(host, port))
	if err != nil {
		return err
	}
	defer l.Close()

	if qr, _ := flags.GetBool("qr-code"); qr {
		if host == "" {
			hostIP, err := oneshotnet.GetSourceIP("", "")
			if err == nil {
				host = hostIP
			}
		}
		output.WriteListeningOnQR(ctx, "http", host, port)
	}

	return r.server.Serve(ctx, l)
}

var ErrTimeout = errors.New("timeout")
