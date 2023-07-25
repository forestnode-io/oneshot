package root

import (
	"fmt"
	"net/http"
	"os"

	"github.com/forestnode-io/oneshot/v2/pkg/configuration"
	oneshothttp "github.com/forestnode-io/oneshot/v2/pkg/net/http"
	"github.com/rs/cors"
)

func (r *rootCommand) configureServer() (string, error) {
	var (
		sConf      = r.config.Server
		timeout    = sConf.Timeout
		allowBots  = sConf.AllowBots
		exitOnFail = sConf.ExitOnFail

		baConf = r.config.BasicAuth
		uname  = baConf.Username
		passwd = baConf.Password

		err error
	)

	var (
		unauthenticatedViewBytes []byte
		unauthenticatedStatus    int
	)
	if uname != "" || (uname != "" && passwd != "") {
		viewPath := baConf.UnauthorizedPage
		if viewPath != "" {
			unauthenticatedViewBytes, err = os.ReadFile(viewPath)
			if err != nil {
				return "", fmt.Errorf("failed to read unauthorized page: %w", err)
			}
		}

		unauthenticatedStatus = baConf.UnauthorizedStatus
	}

	goneHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	})

	var corsMW func(http.Handler) http.Handler
	if copts := corsOptionsFromConfig(&r.config.CORS); copts != nil {
		corsMW = cors.New(*copts).Handler
	}

	noLoginTrigger := baConf.NoDialog
	baMiddleware, baToken, err := oneshothttp.BasicAuthMiddleware(
		unauthenticatedHandler(!noLoginTrigger, unauthenticatedStatus, unauthenticatedViewBytes),
		uname, passwd)
	if err != nil {
		return "", fmt.Errorf("failed to create basic auth middleware: %w", err)
	}

	maxReadSize, err := configuration.ParseSizeString(sConf.MaxReadSize)
	if err != nil {
		return "", fmt.Errorf("failed to parse max read size: %w", err)
	}

	r.server = oneshothttp.NewServer(r.Context(), r.handler, goneHandler, []oneshothttp.Middleware{
		r.middleware.
			Chain(oneshothttp.BlockPrefetch("Safari")).
			Chain(oneshothttp.LimitReaderMiddleware(maxReadSize)).
			Chain(oneshothttp.MiddlewareShim(corsMW)).
			Chain(oneshothttp.BotsMiddleware(allowBots)).
			Chain(baMiddleware),
	}...)
	r.server.TLSCert = sConf.TLSCert
	r.server.TLSKey = sConf.TLSKey
	r.server.Timeout = timeout
	r.server.ExitOnFail = exitOnFail

	return baToken, nil
}
