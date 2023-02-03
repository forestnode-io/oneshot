package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands/root"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()
	ctx = events.WithEvents(ctx)

	status := events.ExitCodeGenericFailure
	defer func() {
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

	ld := os.Getenv("ONESHOT_LOG_DIR")
	if ld == "" {
		if cacheDir, _ := os.UserCacheDir(); cacheDir != "" {
			ld = filepath.Join(cacheDir, "oneshot")
			if err := os.Mkdir(ld, os.ModeDir|0700); err != nil {
				if !os.IsExist(err) {
					ld = ""
				}
			}
		}
	}

	if ld != "" {
		lp := filepath.Join(ld, "oneshot.log")

		logFile, err := os.OpenFile(lp, os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			fmt.Printf("unable to open log file in %s (ONESHOT_LOG_DIR)", ld)
			return
		}
		defer logFile.Close()

		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Llongfile)
	} else {
		log.SetOutput(io.Discard)
	}

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
