package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

var (
	log = zerolog.New(io.Discard)
)

func Logging(ctx context.Context) (context.Context, func(), error) {
	cleanup := func() {}
	logDir := os.Getenv("ONESHOT_LOG_DIR")
	if logDir == "" {
		if cacheDir, _ := os.UserCacheDir(); cacheDir != "" {
			logDir = filepath.Join(cacheDir, "oneshot")
			if err := os.Mkdir(logDir, os.ModeDir|0700); err != nil {
				if !os.IsExist(err) {
					logDir = ""
				}
			}
		}
	}

	var output io.Writer
	if logDir != "" {
		logPath := filepath.Join(logDir, "oneshot.log")
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			return nil, cleanup, fmt.Errorf("unable to open log file in %s (ONESHOT_LOG_DIR)", logDir)
		}
		output = logFile
		cleanup = func() {
			logFile.Close()
		}
	} else {
		output = io.Discard
	}

	if os.Getenv("ONESHOT_LOG_STDERR") != "" {
		output = os.Stderr
	}

	log = zerolog.New(output).With().
		Timestamp().
		Stack().
		Caller().
		Logger()
	ctx = log.WithContext(ctx)
	return ctx, cleanup, nil
}

func Logger() *zerolog.Logger {
	return &log
}