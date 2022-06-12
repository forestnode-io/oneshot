package commands

import (
	"net/http"
	"strings"

	"github.com/raphaelreyna/oneshot/v2/internal/server"
	"github.com/raphaelreyna/oneshot/v2/internal/summary"
)

func botsMiddleware(block bool) server.Middleware {
	if !block {
		return func(hf server.HandlerFunc) server.HandlerFunc {
			return hf
		}
	}
	return func(next server.HandlerFunc) server.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) (*summary.Request, error) {
			// Filter out requests from bots, iMessage, etc. by checking the User-Agent header for known bot headers
			if headers, exists := r.Header["User-Agent"]; exists {
				if isBot(headers) {
					w.WriteHeader(http.StatusOK)
					return nil, nil
				}
			}
			return next(w, r)
		}
	}
}

func authMiddleware(unauthenticated http.HandlerFunc, username, password string) server.Middleware {
	if username == "" && password == "" {
		return func(hf server.HandlerFunc) server.HandlerFunc {
			return hf
		}
	}

	return func(authenticated server.HandlerFunc) server.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) (*summary.Request, error) {
			u, p, ok := r.BasicAuth()
			if !ok {
				unauthenticated(w, r)
				return nil, nil
			}
			// Whichever field is missing is not checked
			if username != "" && username != u {
				unauthenticated(w, r)
				return nil, nil
			}
			if password != "" && password != p {
				unauthenticated(w, r)
				return nil, nil
			}
			return authenticated(w, r)
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
