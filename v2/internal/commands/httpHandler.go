package commands

import (
	"context"
	"net/http"
)

type httpHandlerKey struct{}

func WithHTTPHandlerFuncSetter(ctx context.Context, h *http.HandlerFunc) context.Context {
	return context.WithValue(ctx, httpHandlerKey{}, h)
}

func SetHTTPHandlerFunc(ctx context.Context, h http.HandlerFunc) {
	if hp, ok := ctx.Value(httpHandlerKey{}).(*http.HandlerFunc); ok {
		*hp = h
	}
}
