package root

import (
	"net/http"
	"os"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/rs/cors"
	"github.com/spf13/pflag"
)

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
			Chain(oneshothttp.MiddlewareShim(corsMW)).
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
