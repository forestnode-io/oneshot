package handlers

import (
	"fmt"
	"github.com/jf-tech/iohelper"
	"github.com/raphaelreyna/oneshot/internal/file"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func HandleUpload(file *file.FileWriter, unixEOLNormalization bool, infoLog *log.Logger) func(w http.ResponseWriter, r *http.Request) error {
	// bytes encoding linefeed and carriage-return linefeed
	// used for converting between DOS and UNIX file types
	var (
		lf   = []byte{10}
		crlf = []byte{13, 10}
	)

	// Creating logging messages and functions
	msg := "transfer complete:\n"
	msg += "\tname: %s\n"
	msg += "\tlocation: %s\n"
	msg += "\tsize: %s\n"
	msg += "\tstart time: %s\n"
	msg += "\tduration: %s\n"
	msg += "\trate: %s\n"
	msg += "\tsource: %s\n"

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
		durationTime := duration.Truncate(time.Microsecond).String()

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

		file.ProgressWriter.Write([]byte("\n"))
		iLog(msg, file.Name(), file.GetLocation(),
			sizeString, startTime, durationTime,
			rateString, client)
	}

	regex := regexp.MustCompile(`filename="(.+)"`)

	fileName := func(s string) string {
		subs := regex.FindStringSubmatch(s)
		if len(subs) > 1 {
			return subs[1]
		}
		return ""
	}

	// Define and return the actual handler
	return func(w http.ResponseWriter, r *http.Request) error {
		var (
			src io.Reader
			cl  int64 // content-length
			err error
		)

		// Switch on the type of upload to obtain the appropriate src io.Reader to read data from.
		// Uploads may happen by uploading a file, uploading text from an HTML text box, or straight from the request body
		rct := r.Header.Get("Content-Type")
		switch {
		case strings.Contains(rct, "multipart/form-data"): // User uploaded a file
			reader, err := r.MultipartReader()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return err
			}
			part, err := reader.NextPart()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return err
			}

			cd := part.Header.Get("Content-Disposition")
			if file.Path != "" && file.Name() == "" {
				if fn := fileName(cd); fn != "" {
					if file.Path == "" {
						file.Path = "."
					}
					file.SetName(fn, true)
				}
			}

			src = part

			cl, err = strconv.ParseInt(part.Header.Get("Content-Length"), 10, 64)
			if err != nil {
				cl = 0
			}
		case strings.Contains(rct, "application/x-www-form-urlencoded"): // User uploaded text from HTML text box
			err := r.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return err
			}
			src = strings.NewReader(r.PostForm.Get("oneshotTextUpload"))
			if unixEOLNormalization {
				src = iohelper.NewBytesReplacingReader(src, crlf, lf)
			}
		default: // Could not determine how file upload was initiated, grabbing the request body
			cd := r.Header.Get("Content-Disposition")
			if file.Path != "" && file.Name() == "" {
				if fn := fileName(cd); fn != "" {
					if file.Path == "" {
						file.Path = "."
					}
					file.SetName(fn, true)
				}
			}
			file.MIMEType = rct
			src = r.Body
			cl, err = strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
			if err != nil {
				cl = 0
			}
		}

		// Make sure no other potentially connecting clients may upload a file now (oneshot!)
		file.Lock()
		if err == nil && cl != 0 {
			file.SetSize(cl)
		}
		err = file.Open()
		defer func() {
			file.Close()
		}()
		if err != nil {
			file.Unlock()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		// Start writing the incoming data to disc or stdout while timing how long it takes
		defer r.Body.Close()
		before := time.Now()
		_, err = io.Copy(file, src)
		duration := time.Since(before)
		if err != nil {
			file.Reset()
			file.Unlock()
			return err
		}
		file.Unlock()

		if file.Path != "" {
			printSummary(before, duration, float64(file.GetSize()), r.RemoteAddr)
		}

		return nil
	}
}
