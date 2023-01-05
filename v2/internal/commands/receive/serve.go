package receive

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raphaelreyna/oneshot/v2/internal/api"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
	"github.com/raphaelreyna/oneshot/v2/internal/out/events"
)

func (c *Cmd) ServeHTTP(actx api.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		c._handleGET(w, r)
		return
	}

	actx.Raise(out.NewHTTPRequest(r))

	var (
		src io.ReadCloser
		cl  int64 // content-length
		err error
	)

	// Switch on the type of upload to obtain the appropriate src io.Reader to read data from.
	// Uploads may happen by uploading a file, uploading text from an HTML text box, or straight from the request body
	rct := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(rct, "multipart/form-data"): // User uploaded a file
		src, cl, err = c.readCloserFromMultipartFormData(r)
	case strings.Contains(rct, "application/x-www-form-urlencoded"): // User uploaded text from HTML text box
		src, cl, err = c.readCloserFromApplicationWWWForm(r)
	default: // Could not determine how file upload was initiated, grabbing the request body
		src, cl, err = c.readCloserFromRawBody(r)
	}
	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), err.(*httpError).stat)
		actx.Raise(&events.ClientDisconnected{
			Err: err,
		})
		return
	}

	if c.decodeBase64Output && 0 < cl {
		src = io.NopCloser(base64.NewDecoder(base64.StdEncoding, src))
	}

	c.file.Lock()
	defer c.file.Unlock()

	if fileSize := int(cl); fileSize != 0 {
		// if decoding base64
		if c.decodeBase64Output {
			cl = int64(base64.StdEncoding.DecodedLen(fileSize))
		}
		c.file.SetSize(cl)
	}

	ctx := c.cobraCommand.Context()
	// open the file we are writing to
	if err = c.file.Open(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		actx.Raise(&events.ClientDisconnected{
			Err: err,
		})
		return
	}

	success := false
	cancelProgDisp := out.DisplayProgress(
		&c.file.Progress,
		125*time.Millisecond,
		r.RemoteAddr,
		c.file.GetSize(),
	)
	defer func() {
		cancelProgDisp(success)
	}()

	file, getBufBytes := out.NewBufferedWriter(c.file)
	fileReport := events.File{
		MIME: c.file.MIMEType,
		Size: c.file.GetSize(),
		Name: c.file.ClientProvidedName(),

		TransferStartTime: time.Now(),
	}

	fileReport.TransferSize, err = io.Copy(file, src)
	if err != nil {
		fileReport.TransferEndTime = time.Now()
		actx.Raise(&fileReport)

		cancelProgDisp(false)
		c.file.Reset()

		actx.Raise(&events.ClientDisconnected{
			Err: err,
		})
		return
	}
	c.file.Close()

	fileReport.Path = c.file.Path
	fileReport.Content = getBufBytes
	fileReport.TransferEndTime = time.Now()
	actx.Raise(&fileReport)

	actx.Success()
	success = true
}

func (c *Cmd) _handleGET(w http.ResponseWriter, r *http.Request) {
	withJS := true
	ua := r.Header.Get("User-Agent")
	if strings.Contains(ua, "curl") || strings.Contains(ua, "wget") {
		withJS = false
	}
	c.writeTemplate(w, withJS)
}

func (c *Cmd) ServeExpiredHTTP(_ api.Context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
