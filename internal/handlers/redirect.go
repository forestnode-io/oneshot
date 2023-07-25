package handlers

import (
	srvr "github.com/forestnode-io/oneshot/internal/server"
	"log"
	"net/http"
	"time"
)

func HandleRedirect(url string, statCode int, noBots bool, header http.Header, infoLog *log.Logger) srvr.FailableHandler {
	// Creating logging messages and functions
	msg := "redirect complete:\n"
	msg += "\tstart time: %s\n"
	msg += "\tclient I.P. address: %s\n"
	msg += "\tredirected to: %s\n"
	msg += "\tHTTP status: %d - %s\n"

	var iLog = func(format string, v ...interface{}) {
		if infoLog != nil {
			infoLog.Printf(format, v...)
		}
	}

	var printSummary = func(start time.Time, client string, rt string) {

		startTime := start.Format("15:04:05.000 MST 2 Jan 2006")

		iLog(msg, startTime, client, rt, statCode, http.StatusText(statCode))
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
		// Set any headers added by the user via flags before redirecting
		for key := range header {
			w.Header().Set(key, header.Get(key))
		}
		http.Redirect(w, r, url, statCode)
		printSummary(time.Now(), r.RemoteAddr, url)
		return nil
	}
}
