package http

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"golang.org/x/crypto/bcrypt"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

func (mw Middleware) Chain(m Middleware) Middleware {
	if mw == nil {
		return m
	}
	return func(hf http.HandlerFunc) http.HandlerFunc {
		hf = mw(hf)
		return m(hf)
	}
}

func BotsMiddleware(block bool) Middleware {
	if !block {
		return func(hf http.HandlerFunc) http.HandlerFunc {
			return hf
		}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Filter out requests from bots, iMessage, etc. by checking the User-Agent header for known bot headers
			if headers, exists := r.Header["User-Agent"]; exists {
				if isBot(headers) {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			next(w, r)
		}
	}
}

func BasicAuthMiddleware(unauthenticated http.HandlerFunc, username, password string) (Middleware, *messages.BasicAuth, error) {
	if username == "" && password == "" {
		return func(hf http.HandlerFunc) http.HandlerFunc {
			return hf
		}, nil, nil
	}

	uHash := sha256.Sum256([]byte(username))
	pHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash username or password: %w", err)
	}
	ba := messages.BasicAuth{
		UsernameHash: uHash[:],
		PasswordHash: pHash,
		Token:        uuid.NewString(),
	}

	return func(authenticated http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if token := r.Header.Get("X-HTTPOverWebRTC-Authorization"); token != "" {
				if token == ba.Token {
					authenticated(w, r)
					return
				}
			}

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
			authenticated(w, r)
		}
	}, &ba, nil
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

func LimitReaderMiddleware(limit int64) Middleware {
	if limit == 0 {
		return func(hf http.HandlerFunc) http.HandlerFunc {
			return hf
		}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next(w, r)
		}
	}
}

func MiddlewareShim(mw func(http.Handler) http.Handler) Middleware {
	if mw == nil {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return next
		}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return mw(http.HandlerFunc(next)).ServeHTTP
	}
}
