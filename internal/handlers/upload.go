package handlers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jf-tech/iohelper"
	"github.com/raphaelreyna/oneshot/internal/file"
	srvr "github.com/raphaelreyna/oneshot/internal/server"
)

func HandleUpload(file *file.FileWriter, unixEOLNormalization bool, csrfToken string, infoLog *log.Logger) srvr.FailableHandler {
	// bytes encoding linefeed and carriage-return linefeed
	// used for converting between DOS and UNIX file types
	var (
		lf   = []byte{10}
		crlf = []byte{13, 10}
	)

	// record each attempt to show in the summary report
	type attempt struct {
		address     string
		size        int64
		transferred int64
		start       time.Time
		stop        time.Time
		filename    string
	}
	attempts := []*attempt{}
	attemptsLock := &sync.Mutex{}

	// Creating logging messages and functions
	msg := "transfer complete\ntransfer summary:\n"
	msg += "\tname: %s\n"
	msg += "\tlocation: %s\n"
	msg += "\tsize: %s\n"
	msg += "\tstart time: %s\n"
	msg += "\tduration: %s\n"
	msg += "\trate: %s\n"
	msg += "\tclient address: %s\n"

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

	var printSummary = func(start time.Time, duration time.Duration, fileSize float64, client string) {

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

		if len(attempts) > 0 {
			failedAttempts := "\t\t- start time: %v\n"
			failedAttempts += "\t\t  duration: %v\n"
			failedAttempts += "\t\t  client address: %s\n"
			failedAttempts += "\t\t  file name: %s\n"
			failedAttempts += "\t\t  file size: %v\n"
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
					aTransferredString = fmt.Sprintf("%.3f KB", float64(a.transferred)/kb)
				case a.transferred < gb:
					aTransferredString = fmt.Sprintf("%.3f MB", float64(a.transferred)/mb)
				default:
					aTransferredString = fmt.Sprintf("%.3f GB", float64(a.transferred)/gb)
				}

				// format size
				var aSize interface{} = a.size
				switch {
				case a.size == 0:
					aSize = "not provided by client"
				case a.size < kb:
					aSize = fmt.Sprintf("%d B", a.size)
				case a.size < mb:
					aSize = fmt.Sprintf("%.3f KB", a.size/kb)
				case a.size < gb:
					aSize = fmt.Sprintf("%.3f MB", a.size/mb)
				default:
					aSize = fmt.Sprintf("%.3f GB", a.size/gb)
				}

				// compute the transfer rate of the failed attempt
				aDuration := a.stop.Sub(a.start)
				aRate := float64(a.transferred) / aDuration.Seconds()
				var aRateString string
				switch {
				case aRate < kb:
					aRateString = fmt.Sprintf("%.3f B/s", aRate)
				case aRate < mb:
					aRateString = fmt.Sprintf("%.3f KB/s", aRate/kb)
				case aRate < gb:
					aRateString = fmt.Sprintf("%.3f MB/s", aRate/mb)
				default:
					aRateString = fmt.Sprintf("%.3f GB/s", aRate/gb)
				}

				msg += fmt.Sprintf(failedAttempts, a.start.Format("15:04:05.000 MST 2 Jan 2006"),
					aDuration.Truncate(time.Microsecond),
					a.address, a.filename, aSize,
					aTransferredString, aRateString,
				)

				if i != len(attempts)-1 {
					msg += "\n"
				}
			}
		}

		file.ProgressWriter.Write([]byte("\n"))
		iLog(msg, file.Name(), file.GetLocation(),
			sizeString, startTime, durationTime,
			rateString, client)
	}

	// We need a way to extract the filename (if its given) from the header data
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
		a := &attempt{
			address: r.RemoteAddr,
			start:   time.Now(),
		}
		defer func() {
			a.stop = time.Now()
			attemptsLock.Lock()
			attempts = append(attempts, a)
			attemptsLock.Unlock()
		}()

		// Switch on the type of upload to obtain the appropriate src io.Reader to read data from.
		// Uploads may happen by uploading a file, uploading text from an HTML text box, or straight from the request body
		rct := r.Header.Get("Content-Type")
		switch {
		case strings.Contains(rct, "multipart/form-data"): // User uploaded a file
			// Check for csrf token if we care to
			if csrfToken != "" && r.Header.Get("X-CSRF-Token") != csrfToken {
				err := errors.New("Invalid CSRF token")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return err
			}

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
			foundCSRFToken := false
			// Assume we found the CSRF token if the user doesn't care to use one
			if csrfToken == "" {
				foundCSRFToken = true
			}

			// Look for the CSRF token in the header
			if r.Header.Get("X-CSRF-Token") == csrfToken && csrfToken != "" {
				foundCSRFToken = true
			}

			err := r.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return err
			}

			// If we havent found the CSRF token yet, look for it in the parsed form data
			if !foundCSRFToken && r.PostFormValue("csrf-token") != csrfToken {
				err := errors.New("Invalid CSRF token")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return err
			}

			src = strings.NewReader(r.PostForm.Get("oneshotTextUpload"))
			if unixEOLNormalization {
				src = iohelper.NewBytesReplacingReader(src, crlf, lf)
			}
		default: // Could not determine how file upload was initiated, grabbing the request body
			// Check for csrf token if we care to
			if csrfToken != "" && r.Header.Get("X-CSRF-Token") != csrfToken {
				err := errors.New("Invalid CSRF token")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return err
			}

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

		a.filename = file.Name()
		// Make sure no other potentially connecting clients may upload a file now (oneshot!)
		file.Lock()
		if err == nil && cl != 0 {
			file.SetSize(cl)
			a.size = cl
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
		stop := time.Now()
		a.transferred = file.GetProgress()
		a.stop = stop
		if err != nil {
			file.Reset()
			file.Unlock()
			return err
		}
		file.Unlock()

		if file.Path != "" {
			printSummary(before, stop.Sub(before), float64(file.GetSize()), r.RemoteAddr)
		}

		return nil
	}
}
