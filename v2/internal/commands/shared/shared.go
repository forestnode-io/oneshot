package shared

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func HeaderFromStringSlice(s []string) http.Header {
	h := make(http.Header)
	if s == nil {
		return h
	}

	for _, hs := range s {
		var (
			parts = strings.SplitN(hs, "=", 1)
			k     = parts[0]
			v     = ""
		)

		if len(parts) == 2 {
			v = parts[1]
		}

		var vs = h[k]
		if vs == nil {
			vs = make([]string, 0)
		}
		vs = append(vs, v)
		h[k] = vs
	}

	return h
}

type closerKey struct{}

func WithClosers(ctx context.Context, closers *[]io.Closer) context.Context {
	return context.WithValue(ctx, closerKey{}, closers)
}

func MarkForClose(ctx context.Context, closer io.Closer) {
	if closers, ok := ctx.Value(closerKey{}).(*[]io.Closer); ok {
		*closers = append(*closers, closer)
	}
}
