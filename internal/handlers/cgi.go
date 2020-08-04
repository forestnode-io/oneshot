package handlers

import (
	ezcgi "github.com/raphaelreyna/ez-cgi/pkg/cgi"
	"github.com/raphaelreyna/oneshot/internal/server"
	"log"
	"net/http"
	"strings"
	"time"
)

func HandleCGI(handler *ezcgi.Handler, name, mime string, infoLog *log.Logger) func(w http.ResponseWriter, r *http.Request) error {
	msg := "transfer complete:\n"
	msg += "\tname: %s\n"
	if mime != "" {
		msg += "\tMIME type: %s\n"
	}
	msg += "\tstart time: %s\n"
	msg += "\tduration: %s\n"
	msg += "\tdestination: %s\n"

	var iLog = func(format string, v ...interface{}) {
		if infoLog != nil {
			infoLog.Printf(format, v...)
		}
	}

	var printSummary = func(start time.Time,
		duration time.Duration, client string) {

		startTime := start.Format("15:04:05.000 MST 2 Jan 2006")
		durationTime := duration.Truncate(time.Millisecond).String()

		if mime != "" {
			iLog(msg, name, mime, startTime, durationTime, client)
		} else {
			iLog(msg, name, startTime, durationTime, client)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) error {
		// Filter out requests from bots, iMessage, etc.
		if headers, exists := r.Header["User-Agent"]; exists {
			for _, header := range headers {
				isBot := strings.Contains(header, "bot")
				if !isBot {
					isBot = strings.Contains(header, "Bot")
				}
				if !isBot {
					isBot = strings.Contains(header, "facebookexternalhit")
				}
				if isBot {
					w.WriteHeader(http.StatusOK)
					return server.OKNotDoneErr
				}
			}
		}
		iLog("connected: %s", r.RemoteAddr)
		before := time.Now()
		handler.ServeHTTP(w, r)
		duration := time.Since(before)
		printSummary(before, duration, r.RemoteAddr)
		return nil
	}
}
