package handlers

import (
	ezcgi "github.com/raphaelreyna/ez-cgi/pkg/cgi"
	srvr "github.com/oneshot-uno/oneshot/internal/server"
	"log"
	"net/http"
	"time"
)

func HandleCGI(handler *ezcgi.Handler, name, mime string, noBots bool, infoLog *log.Logger) srvr.FailableHandler {
	// Creating logging messages and functions
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

	// Define and return the actual handler
	return func(w http.ResponseWriter, r *http.Request) error {
		// Filter out requests from bots, iMessage, etc. by checking the User-Agent header for known bot headers
		if headers, exists := r.Header["User-Agent"]; exists && noBots {
			if isBot(headers) {
				w.WriteHeader(http.StatusOK)
				return srvr.OKNotDoneErr
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
