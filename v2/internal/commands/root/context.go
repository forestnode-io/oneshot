package root

import (
	"context"
)

type fileGCKey struct{}

func withFileGarbageCollection(ctx context.Context, files *[]string) context.Context {
	return context.WithValue(ctx, fileGCKey{}, files)
}

func markFilesAsGarbage(ctx context.Context, filePaths ...string) {
	if files, ok := ctx.Value(fileGCKey{}).(*[]string); ok {
		*files = append(*files, filePaths...)
	}
}
