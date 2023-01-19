package root

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
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

	defer output.Wait(ctx)

	if err := r.configureServer(flags); err != nil {
		return err
	}

	return r.listenAndServe(ctx, flags)
}

func (r *rootCommand) configureServer(flags *pflag.FlagSet) error {
	var (
		jopts, _               = flags.GetString("json")
		timeout, _             = flags.GetDuration("timeout")
		allowBots, _           = flags.GetBool("allow-bots")
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

	// r.handler should have been set by now by one of the subcommands
	r.server.HandlerFunc = r.handler
	r.server.TLSCert = tlsCert
	r.server.TLSKey = tlsKey
	r.server.Timeout = timeout
	r.server.BufferRequestBody = strings.Contains(jopts, "include-body")
	r.server.Middleware = []oneshothttp.Middleware{r.middleware.
		Chain(oneshothttp.BotsMiddleware(allowBots)).
		Chain(oneshothttp.BasicAuthMiddleware(unauthenticatedHandler, uname, passwd)),
	}

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

	return r.server.serve(ctx, 1, l)
}

var ErrTimeout = errors.New("timeout")

type server struct {
	HandlerFunc       http.HandlerFunc
	TLSCert, TLSKey   string
	Middleware        []oneshothttp.Middleware
	BufferRequestBody bool
	Timeout           time.Duration

	http.Server
}

func (s *server) serve(ctx context.Context, queueSize int64, l net.Listener) error {
	// demux the handler and apply middleware
	httpHandler, cancelDemux := oneshothttp.Demux(queueSize, s.HandlerFunc)
	defer cancelDemux()
	for _, mw := range s.Middleware {
		httpHandler = mw(httpHandler)
	}

	// create the router and register the demuxed handler
	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(httpHandler)
	s.Handler = r

	if 0 < s.Timeout {
		l = oneshotnet.NewListenerTimer(l, s.Timeout)
	}

	errs := make(chan error, 1)
	go func() {
		if s.TLSKey != "" {
			errs <- s.Server.ServeTLS(l, s.TLSCert, s.TLSKey)
		} else {
			errs <- s.Server.Serve(l)
		}
	}()

	<-ctx.Done()

	//shutdown gracefully
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	_ = s.Server.Shutdown(ctx)
	cancel()

	err := <-errs
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	return err
}
