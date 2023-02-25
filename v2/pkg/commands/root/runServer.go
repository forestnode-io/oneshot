package root

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/ice"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/server"
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
		ctx   = cmd.Context()
		flags = cmd.Flags()
	)

	defer func() {
		events.Stop(ctx)
		output.Wait(ctx)
	}()

	if r.handler == nil {
		return nil
	}

	if err := r.configureServer(flags); err != nil {
		return err
	}

	webrtc, _ := flags.GetBool("webrtc")
	webRTCSignallingDir, _ := flags.GetString("webrtc-signalling-dir")
	if webrtc || webRTCSignallingDir != "" {
		if err := r.configureWebRTC(flags); err != nil {
			return err
		}

		var signaller sdp.ServerSignaller
		if output.IsTTYForContentOnly(ctx) {
			if webRTCSignallingDir == "" {
				return errors.New("signalling directory must be set (--webrtc-signalling-dir) when serving from stdin or to stdout")
			}
			signaller = sdp.NewFileServerSignaller(webRTCSignallingDir)
		} else if webRTCSignallingDir != "" {
			signaller = sdp.NewFileServerSignaller(webRTCSignallingDir)
		} else {
			signaller = sdp.NewTTYServerSignaller()
		}

		a := server.Server{
			Handler: http.HandlerFunc(r.server.ServeHTTP),
			Config:  r.webrtcConfig,
		}
		go func() {
			if err := signaller.Start(ctx, &a); err != nil {
				log.Fatal(err)
			}
		}()
	}

	return r.listenAndServe(ctx, flags)
}

func (r *rootCommand) configureServer(flags *pflag.FlagSet) error {
	var (
		timeout, _    = flags.GetDuration("timeout")
		allowBots, _  = flags.GetBool("allow-bots")
		exitOnFail, _ = flags.GetBool("exit-on-fail")
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

	sfa := flags.Lookup("max-read-size").Value.(*commands.SizeFlagArg)

	noLoginTrigger, _ := flags.GetBool("dont-trigger-login")
	r.server = oneshothttp.NewServer(r.Context(), r.handler, goneHandler, []oneshothttp.Middleware{
		r.middleware.
			Chain(oneshothttp.LimitReaderMiddleware(int64(*sfa))).
			Chain(middlewareShim(corsMW)).
			Chain(oneshothttp.BotsMiddleware(allowBots)).
			Chain(oneshothttp.BasicAuthMiddleware(
				unauthenticatedHandler(!noLoginTrigger, unauthenticatedStatus, unauthenticatedViewBytes),
				uname, passwd)),
	}...)
	r.server.TLSCert = tlsCert
	r.server.TLSKey = tlsKey
	r.server.Timeout = timeout
	r.server.ExitOnFail = exitOnFail

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

func (r *rootCommand) configureWebRTC(flags *pflag.FlagSet) error {
	urls, _ := flags.GetStringSlice("webrtc-ice-servers")
	if len(urls) == 0 {
		urls = ice.STUNServerURLS
	}

	r.webrtcConfig = &webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: urls,
			},
		},
	}

	return nil
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
{{corsFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "NAT Traversal Flags:" | indent 4 }}
{{ "WebRTC Flags:" | indent 8 }}
{{webrtcFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 12}}{{if eq .Name "oneshot" }}

Use "oneshot [command] --help" for more information about a command.{{end}}
`

const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
If you encounter any bugs or have any questions or suggestions, please open an issue at:
https://github.com/raphaelreyna/oneshot/issues/new/choose
`
