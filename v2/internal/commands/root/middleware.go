package root

import (
	"net/http"
	"strings"

	"github.com/raphaelreyna/oneshot/v2/internal/api"
	"github.com/raphaelreyna/oneshot/v2/internal/server"
)

func botsMiddleware(block bool) server.Middleware {
	if !block {
		return func(hf api.HTTPHandler) api.HTTPHandler {
			return hf
		}
	}
	return func(next api.HTTPHandler) api.HTTPHandler {
		return func(actx api.Context, w http.ResponseWriter, r *http.Request) {
			// Filter out requests from bots, iMessage, etc. by checking the User-Agent header for known bot headers
			if headers, exists := r.Header["User-Agent"]; exists {
				if isBot(headers) {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			next(actx, w, r)
		}
	}
}

func authMiddleware(unauthenticated http.HandlerFunc, username, password string) server.Middleware {
	if username == "" && password == "" {
		return func(hf api.HTTPHandler) api.HTTPHandler {
			return hf
		}
	}

	return func(authenticated api.HTTPHandler) api.HTTPHandler {
		return func(actx api.Context, w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok {
				unauthenticated(w, r)
				return
			}
			// Whichever field is missing is not checked
			if username != "" && username != u {
				unauthenticated(w, r)
				return
			}
			if password != "" && password != p {
				unauthenticated(w, r)
				return
			}
			authenticated(actx, w, r)
		}
	}
}

// botHeaders are the known User-Agent header values in use by bots / machines
var botHeaders []string = []string{
	"bot",
	"Bot",
	"facebookexternalhit",
}

func isBot(headers []string) bool {
	for _, header := range headers {
		for _, botHeader := range botHeaders {
			if strings.Contains(header, botHeader) {
				return true
			}
		}
	}

	return false
}
