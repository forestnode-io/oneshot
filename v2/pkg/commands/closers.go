package commands

import (
	"context"
	"io"
)

type closerKey struct{}

func WithClosers(ctx context.Context, closers *[]io.Closer) context.Context {
	return context.WithValue(ctx, closerKey{}, closers)
}

func MarkForClose(ctx context.Context, closer io.Closer) {
	if closers, ok := ctx.Value(closerKey{}).(*[]io.Closer); ok {
		*closers = append(*closers, closer)
	}
}
