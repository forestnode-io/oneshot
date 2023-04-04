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
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/log"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	var (
		status = events.ExitCodeGenericFailure
		err    error
	)

	ctx, cleanupLogging, err := log.Logging(context.Background())
	if err != nil {
		panic(err)
	}
	defer cleanupLogging()

	ctx = events.WithEvents(ctx)
	ctx, err = output.WithOutput(ctx)
	if err != nil {
		fmt.Printf("error setting up output: %s\n", err.Error())
		return
	}

	defer func() {
		output.RestoreCursor(ctx)
		if r := recover(); r != nil {
			panic(r)
		} else {
			if ec := events.GetExitCode(ctx); -1 < ec {
				status = ec
			}
			os.Exit(status)
		}
	}()

	sigs := []os.Signal{
		os.Interrupt,
		os.Kill,
	}
	if _, isUnix := unixOS[runtime.GOOS]; isUnix {
		sigs = append(sigs, syscall.SIGINT, syscall.SIGHUP)
	}
	ctx, cancel := signal.NotifyContext(ctx, sigs...)
	defer cancel()

	if err := root.ExecuteContext(ctx); err == nil {
		status = events.ExitCodeSuccess
	}
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
