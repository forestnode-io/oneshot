package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/internal/file"
	srvr "github.com/raphaelreyna/oneshot/internal/server"
	"sync"
)

func HandleDownload(file *file.FileReader, download, noBots bool, header http.Header, infoLog *log.Logger) srvr.FailableHandler {
	// Creating logging messages and functions
	msg := "transfer complete\ntransfer summary:\n"
	msg += "\tfile name: %s\n"
	msg += "\tMIME type: %s\n"
	msg += "\tfile size: %s\n"
	msg += "\tstart time: %s\n"
	msg += "\tduration: %s\n"
	msg += "\trate: %s\n"
	msg += "\tclient address: %s\n"

	// record each attempt to show in the summary report
	type attempt struct {
		address string
		transferred int64
		start time.Time
		stop time.Time
	}
	attempts := []*attempt{}
	attemptsLock := &sync.Mutex{}

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

		if len(attempts) > 0 {
			failedAttempts := "\t\t- start time: %v\n"
			failedAttempts += "\t\t  duration: %v\n"
			failedAttempts += "\t\t  client address: %s\n"
			failedAttempts += "\t\t  transferred: %v\n"
			failedAttempts += "\t\t  transfer rate: %v\n"

			// add failed attempts header
			msg += "\tfailed attempts:\n"

			// loop over failed attempts and them to the msg string
			for i, a := range attempts {
				// format transfer
				aTransferredString := ""
				switch {
				case a.transferred < kb:
					aTransferredString = fmt.Sprintf("%d B", a.transferred)
				case a.transferred < mb:
					aTransferredString = fmt.Sprintf("%.3f KB", float64(a.transferred) / kb)
				case a.transferred < gb:
					aTransferredString = fmt.Sprintf("%.3f MB", float64(a.transferred) / mb)
				default:
					aTransferredString = fmt.Sprintf("%.3f GB", float64(a.transferred) / gb)
				}

				// compute the transfer rate of the failed attempt
				aDuration := a.stop.Sub(a.start)
				aRate := float64(a.transferred) / aDuration.Seconds()
				var aRateString string
				switch {
				case aRate < kb:
					aRateString = fmt.Sprintf("%.3f B/s", aRate)
				case aRate < mb:
					aRateString = fmt.Sprintf("%.3f KB/s", aRate / kb)
				case aRate < gb:
					aRateString = fmt.Sprintf("%.3f MB/s", aRate / mb)
				default:
					aRateString = fmt.Sprintf("%.3f GB/s", aRate / gb)
				}

				msg += fmt.Sprintf(failedAttempts, a.start.Format("15:04:05.000 MST 2 Jan 2006"),
					aDuration.Truncate(time.Microsecond),
					a.address, aTransferredString, aRateString,
				)

				if i != len(attempts)-1 {
					msg += "\n"
				}
			}
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
				return srvr.OKNotDoneErr
			}
		}

		a := &attempt{
			address: r.RemoteAddr,
			start: time.Now(),
		}
		defer func() {
			a.stop = time.Now()
			attemptsLock.Lock()
			attempts = append(attempts, a)
			attemptsLock.Unlock()
		}()

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
		stop := time.Now()
		a.transferred = file.GetProgress()
		a.stop = stop
		if err != nil {
			return err
		}

		// Let the user know how things went
		printSummary(before, stop.Sub(before), float64(file.Size()), r.RemoteAddr)

		return nil
	}
}
