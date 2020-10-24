package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/internal/file"
	"github.com/raphaelreyna/oneshot/internal/server"
)

func HandleDownload(file *file.FileReader, download, noBots bool, header http.Header, infoLog *log.Logger) func(w http.ResponseWriter, r *http.Request) error {
	// Creating logging messages and functions
	msg := "transfer complete:\n"
	msg += "\tname: %s\n"
	msg += "\tMIME type: %s\n"
	msg += "\tsize: %s\n"
	msg += "\tstart time: %s\n"
	msg += "\tduration: %s\n"
	msg += "\trate: %s\n"
	msg += "\tdestination: %s\n"

	const (
		kb = 1000
		mb = kb * 1000
		gb = mb * 1000
	)

	var iLog = func(format string, v ...interface{}) {
		if infoLog != nil {
			infoLog.Printf(format, v...)
		}
	}

	var printSummary = func(start time.Time,
		duration time.Duration, fileSize float64,
		client string) {

		var sizeString string
		var size float64
		rate := fileSize / duration.Seconds()

		startTime := start.Format("15:04:05.000 MST 2 Jan 2006")
		durationTime := duration.Truncate(time.Millisecond).String()

		// Create the size string using appropriate units: B, KB, MB, and GB
		switch {
		case fileSize < kb:
			sizeString = fmt.Sprintf("%d B", int64(fileSize))
		case fileSize < mb:
			size = fileSize / kb
			sizeString = fmt.Sprintf("%.3f KB", size)
		case fileSize < gb:
			size = fileSize / mb
			sizeString = fmt.Sprintf("%.3f MB", size)
		default:
			size = fileSize / gb
			sizeString = fmt.Sprintf("%.3f GB", size)
		}

		// Create the size string using appropriate units: B/s, KB/s, MB/s, and GB/s
		var rateString string
		switch {
		case rate < kb:
			rateString = fmt.Sprintf("%.3f B/s", rate)
		case rate < mb:
			rate = rate / kb
			rateString = fmt.Sprintf("%.3f KB/s", rate)
		case rate < gb:
			rate = rate / mb
			rateString = fmt.Sprintf("%.3f MB/s", rate)
		default:
			rate = rate / gb
			rateString = fmt.Sprintf("%.3f GB/s", rate)
		}

		mimeType := file.MimeType
		if ct := header.Get("Content-Type"); ct != "" {
			mimeType = ct
		}

		iLog(msg,
			file.Name, mimeType, sizeString,
			startTime, durationTime,
			rateString, client)
	}

	// Define and return the actual handler
	return func(w http.ResponseWriter, r *http.Request) error {
		// Filter out requests from bots, iMessage, etc. by checking the User-Agent header for known bot headers
		if headers, exists := r.Header["User-Agent"]; exists && noBots {
			if isBot(headers) {
				w.WriteHeader(http.StatusOK)
				return server.OKNotDoneErr
			}
		}

		// Client is not a bot so show request info and open file to get file info for HTTP headers
		iLog("connected: %s", r.RemoteAddr)
		err := file.Open()
		defer func() {
			file.Reset()
			file.Close()
		}()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		// Are we triggering a file download on the users browser?
		if download {
			w.Header().Set("Content-Disposition",
				fmt.Sprintf("attachment;filename=%s", file.Name),
			)
		}

		// Set standard Content headers
		w.Header().Set("Content-Type", file.MimeType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", file.Size()))

		// Set any additional headers added by the user via flags
		for key := range header {
			w.Header().Set(key, header.Get(key))
		}

		// Start writing the file data to the client while timing how long it takes
		before := time.Now()
		_, err = io.Copy(w, file)
		duration := time.Since(before)
		if err != nil {
			return err
		}

		// Let the user know how things went
		printSummary(before, duration, float64(file.Size()), r.RemoteAddr)

		return nil
	}
}
