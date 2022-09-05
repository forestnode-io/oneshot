package server

import (
	"github.com/raphaelreyna/oneshot/v2/internal/api"
)

type Middleware func(api.HTTPHandler) api.HTTPHandler

func (mw Middleware) Chain(m Middleware) Middleware {
	if mw == nil {
		return m
	}
	return func(hf api.HTTPHandler) api.HTTPHandler {
		hf = mw(hf)
		return m(hf)
	}
}
