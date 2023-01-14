package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands/root"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	sigs := []os.Signal{
		os.Interrupt,
		os.Kill,
	}
	if _, isUnix := unixOS[runtime.GOOS]; isUnix {
		sigs = append(sigs, syscall.SIGINT, syscall.SIGHUP)
	}
	ctx, cancel := signal.NotifyContext(ctx, sigs...)
	defer cancel()

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

// copied from https://github.com/golang/go/blob/ebb572d82f97d19d0016a49956eb1fddc658eb76/src/go/build/syslist.go#L38
var unixOS = map[string]struct{}{
	"aix":       {},
	"android":   {},
	"darwin":    {},
	"dragonfly": {},
	"freebsd":   {},
	"hurd":      {},
	"illumos":   {},
	"ios":       {},
	"linux":     {},
	"netbsd":    {},
	"openbsd":   {},
	"solaris":   {},
}
