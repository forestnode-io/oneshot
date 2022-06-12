package server

import (
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/internal/summary"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) (*summary.Request, error)

func newHandlerFunc(hf http.HandlerFunc, includeSummary bool) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) (*summary.Request, error) {
		var sr *summary.Request
		if includeSummary {
			sr = summary.NewRequest(r)
		}
		hf(w, r)
		return sr, nil
	}
}

type Middleware func(HandlerFunc) HandlerFunc

func (mw Middleware) Chain(m Middleware) Middleware {
	if mw == nil {
		return m
	}
	return func(hf HandlerFunc) HandlerFunc {
		hf = mw(hf)
		return m(hf)
	}
}
