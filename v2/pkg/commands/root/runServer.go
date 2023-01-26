package root

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
	"github.com/rs/cors"
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

	defer func() {
		events.Stop(ctx)
		output.Wait(ctx)
	}()

	return r.listenAndServe(ctx, flags)
}

func (r *rootCommand) configureServer(flags *pflag.FlagSet) error {
	var (
		timeout, _   = flags.GetDuration("timeout")
		allowBots, _ = flags.GetBool("allow-bots")
	)

	uname, passwd, err := usernamePassword(flags)
	if err != nil {
		return err
	}

	var (
		unauthenticatedViewBytes []byte
		unauthenticatedStatus    int
	)
	if uname != "" || (uname != "" && passwd != "") {
		viewPath, _ := flags.GetString("unauthenticated-view")
		if viewPath != "" {
			unauthenticatedViewBytes, err = os.ReadFile(viewPath)
			if err != nil {
				return err
			}
		}

		unauthenticatedStatus, _ = flags.GetInt("unauthenticated-status")
	}

	tlsCert, tlsKey, err := tlsCertAndKey(flags)
	if err != nil {
		return err
	}

	goneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	})

	var corsMW func(http.Handler) http.Handler
	if copts := corsOptionsFromFlagSet(flags); copts != nil {
		corsMW = cors.New(*copts).Handler
	}

	r.server = oneshothttp.NewServer(r.Context(), r.handler, goneHandler, []oneshothttp.Middleware{
		r.middleware.
			Chain(middlewareShim(corsMW)).
			Chain(oneshothttp.BotsMiddleware(allowBots)).
			Chain(oneshothttp.BasicAuthMiddleware(
				unauthenticatedHandler(unauthenticatedStatus, unauthenticatedViewBytes),
				uname, passwd)),
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

func unauthenticatedHandler(statCode int, content []byte) http.HandlerFunc {
	if statCode == 0 {
		statCode = http.StatusUnauthorized
	}
	if content == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statCode)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statCode)
		_, _ = w.Write(content)
	}
}

func middlewareShim(mw func(http.Handler) http.Handler) oneshothttp.Middleware {
	if mw == nil {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return next
		}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return mw(http.HandlerFunc(next)).ServeHTTP
	}
}

const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if and .HasAvailableLocalFlags (ne .Name "oneshot")}}

Flags:
{{.LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces}}{{end}}

Global Flags:
{{ "Output flags:" | indent 4}}
{{flags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "Server Flags:" | indent 4 }}
{{serverFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "Basic Authentication Flags:" | indent 4 }}
{{basicAuthFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "CORS Flags:" | indent 4 }}
{{corsFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}{{if eq .Name "oneshot" }}

Use "oneshot [command] --help" for more information about a command.{{end}}
`

const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
If you encounter any bugs or have any questions or suggestions, please open an issue at:
https://github.com/raphaelreyna/oneshot/issues/new/choose
`
