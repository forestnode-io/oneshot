package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/raphaelreyna/oneshot/internal/file"
	"github.com/raphaelreyna/oneshot/internal/server"
)

func HandleDownload(file *file.FileReader, download bool, header http.Header,
	infoLog *log.Logger) func(w http.ResponseWriter, r *http.Request) error {
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
		err := file.Open()
		defer func() {
			file.Reset()
			file.Close()
		}()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if download {
			w.Header().Set("Content-Disposition",
				fmt.Sprintf("attachment;filename=%s", file.Name),
			)
		}
		w.Header().Set("Content-Type", file.MimeType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", file.Size()))

		for key := range header {
			w.Header().Set(key, header.Get(key))
		}

		before := time.Now()
		_, err = io.Copy(w, file)
		duration := time.Since(before)
		if err != nil {
			return err
		}

		printSummary(before, duration, float64(file.Size()), r.RemoteAddr)
		return nil
	}
}
