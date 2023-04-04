package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands/root"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/log"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/raphaelreyna/oneshot/v2/pkg/sys"
)

func main() {
	var (
		status = events.ExitCodeGenericFailure
		err    error
	)

	//lint:ignore SA1019 the issues that plague this implementation are not relevant to this project
	rand.Seed(time.Now().UnixNano())

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
	if sys.RunningOnUNIX() {
		sigs = append(sigs, syscall.SIGINT, syscall.SIGHUP)
	}
	ctx, cancel := signal.NotifyContext(ctx, sigs...)
	defer cancel()

	if err := root.ExecuteContext(ctx); err == nil {
		status = events.ExitCodeSuccess
	}
}
