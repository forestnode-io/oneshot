package handlers

import (
	"fmt"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"io"
	"log"
	"net/http"
	"time"
)

func HandleSend(file *server.File, download bool,
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

		iLog(msg,
			file.Name, file.MimeType, sizeString,
			startTime, durationTime,
			rateString, client)
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		iLog("client connected: %s\n", r.RemoteAddr)

		err := file.Open()
		defer func() {
			file.ResetReader()
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
