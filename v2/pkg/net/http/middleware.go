package http

import (
	"net/http"
	"strings"
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

func Demux(queueSize int64, next http.HandlerFunc) (http.HandlerFunc, func()) {
	type _wr struct {
		w    http.ResponseWriter
		r    *http.Request
		done func()
	}

	requestsQueue := make(chan _wr, queueSize)

	go func() {
		for wr := range requestsQueue {
			next(wr.w, wr.r)
			wr.done()
		}
	}()

	mw := func(w http.ResponseWriter, r *http.Request) {
		doneChan := make(chan struct{})
		wr := _wr{
			w: w,
			r: r,
			done: func() {
				close(doneChan)
			},
		}

		requestsQueue <- wr
		<-doneChan
	}

	return mw, func() {
		close(requestsQueue)
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

func BasicAuthMiddleware(unauthenticated http.HandlerFunc, username, password string) Middleware {
	if username == "" && password == "" {
		return func(hf http.HandlerFunc) http.HandlerFunc {
			return hf
		}
	}

	return func(authenticated http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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
