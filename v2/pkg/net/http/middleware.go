package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

func BasicAuthMiddleware(unauthenticated http.HandlerFunc, username, password string) (Middleware, string, error) {
	if username == "" && password == "" {
		return func(hf http.HandlerFunc) http.HandlerFunc {
			return hf
		}, "", nil
	}

	baToken := uuid.NewString()

	return func(authenticated http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if token := r.Header.Get("X-HTTPOverWebRTC-Authorization"); token != "" {
				if token == baToken {
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
	}, baToken, nil
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

// BlockPrefetch blocks prefetching of the page by reloading the page with a short-lived cookie
// that's only set if the page is visible. This is to prevent browsers and bots from prefetching.
// Some have recently started not marking the request as a prefetch, so this is a last resort.
func BlockPrefetch(userAgentKeys ...string) Middleware {
	bpCookieName := "block-prefetch"
	preClient := `<!DOCTYPE html>
<html>
<head>
</head>
<body>
<script>
	if (document.visibilityState === 'visible') {
		document.cookie = '%s=%d' + '; max-age=1; path=/';
		window.location.reload();
	}
</script>
</body>
</html>`
	writePreClient := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		payload := fmt.Sprintf(preClient, bpCookieName, time.Now().UnixNano())
		_, _ = w.Write([]byte(payload))
	}
	cookieIsValid := func(cookie *http.Cookie) bool {
		if cookie == nil {
			return false
		}
		cv, err := strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			return false
		}
		if time.Second < time.Since(time.Unix(0, cv)) {
			return false
		}
		return true
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next(w, r)
				return
			}

			if r.Header.Get("X-HTTPOverWebRTC") == "true" {
				next(w, r)
				return
			}

			userAgent := r.Header.Get("User-Agent")
			for _, key := range userAgentKeys {
				if strings.Contains(userAgent, key) {
					cookie, err := r.Cookie("block-prefetch")
					if err != nil || cookie == nil {
						writePreClient(w)
						return
					}
					if cookieIsValid(cookie) {
						next(w, r)
						return
					}
				}
			}
			next(w, r)
		}
	}
}
