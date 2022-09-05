package main

import (
	"context"
	"fmt"
	"os"

	"github.com/raphaelreyna/oneshot/v2/internal/commands/root"
)

func main() {
	ctx := context.Background()
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
