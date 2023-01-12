package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands/root"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
