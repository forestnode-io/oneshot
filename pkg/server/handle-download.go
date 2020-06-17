package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (s *Server) handleDownload(file *File) http.HandlerFunc {
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
			rateString = fmt.Sprintf("%.3f MB/s", rate)
		}

		s.infoLog(msg,
			file.Name, file.MimeType, sizeString,
			startTime, durationTime,
			rateString, client)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		file.Lock()
		if file.RequestCount() > 0 {
			file.Requested()
			file.Unlock()
			w.WriteHeader(http.StatusGone)
			return
		}
		file.Requested()
		file.Unlock()

		// Stop() method needs to run on seperate goroutine.
		// Otherwise, we deadlock since http.Server.Shutdown()
		// wont return until this function returns.
		defer func() {
			go s.Stop(context.Background())
		}()

		s.infoLog("client connected: %s\n", r.RemoteAddr)

		err := file.Open()
		defer file.Close()
		if err != nil {
			s.err = err
			s.internalError("error while opening file: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if s.Download {
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
			s.err = err
			return
		}

		printSummary(before, duration, float64(file.Size()), r.RemoteAddr)
	}
}
