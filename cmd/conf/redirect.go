package conf

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/forestnode-io/oneshot/internal/handlers"
	"github.com/forestnode-io/oneshot/internal/server"
	"net/url"
)

func (c *Conf) setupRedirectRoute(args []string, srvr *server.Server) (*server.Route, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing redirect URL")
	}

	// Check if status code is valid
	if http.StatusText(c.RedirectStatus) == "" {
		return nil, fmt.Errorf("invalid HTTP status code: %d", c.RedirectStatus)
	}

	u, err := url.Parse(args[0])
	if err != nil { return nil, err }
	if u.Scheme == "" {
		u.Scheme = "http"
	}

	route := &server.Route{
		Pattern: "/",
		DoneHandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("gone"))
		},
	}
	if c.ExitOnFail {
		route.MaxRequests = 1
	} else {
		route.MaxOK = 1
	}

	header := http.Header{}
	for _, rh := range c.RawHeaders {
		parts := strings.SplitN(rh, ":", 2)
		if len(parts) < 2 {
			err := fmt.Errorf("invalid header: %s", rh)
			return nil, err
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		header.Set(k, v)
	}

	route.HandlerFunc = handlers.HandleRedirect(u.String(), c.RedirectStatus, !c.AllowBots, header, srvr.InfoLog)

	return route, nil
}
