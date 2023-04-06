package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
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
		lj := lumberjack.Logger{
			Filename: logPath,
			MaxSize:  500, // megabytes
		}
		output = &lj
		cleanup = func() {
			lj.Close()
		}
	} else {
		output = io.Discard
	}

	if os.Getenv("ONESHOT_LOG_STDERR") != "" {
		output = os.Stderr
	}

	var (
		levelString = os.Getenv("ONESHOT_LOG_LEVEL")
		level       = zerolog.InfoLevel
		err         error
	)
	if levelString != "" {
		level, err = zerolog.ParseLevel(levelString)
		if err != nil {
			return ctx, cleanup, fmt.Errorf("unable to parse log level from ONESHOT_LOG_LEVEL: %s", err.Error())
		}
	}

	logContext := zerolog.New(output).
		Level(level).
		With().
		Timestamp()
	if level == zerolog.DebugLevel {
		logContext = logContext.
			Stack().
			Caller()
	}

	log = logContext.Logger()

	ctx = log.WithContext(ctx)
	return ctx, cleanup, nil
}

func Logger() *zerolog.Logger {
	return &log
}
