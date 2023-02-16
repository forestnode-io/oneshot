package receive

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
)

func (c *Cmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := c.cobraCommand.Context()

	if r.Method == "GET" {
		c._handleGET(w, r)
		return
	}

	events.Raise(ctx, output.NewHTTPRequest(r))

	var (
		rb  *requestBody
		err error
	)

	// Switch on the type of upload to obtain the appropriate src io.Reader to read data from.
	// Uploads may happen by uploading a file, uploading text from an HTML text box, or straight from the request body
	rct := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(rct, "multipart/form-data"): // User uploaded a file
		rb, err = c.readCloserFromMultipartFormData(r)
	case strings.Contains(rct, "application/x-www-form-urlencoded"): // User uploaded text from HTML text box
		rb, err = c.readCloserFromApplicationWWWForm(r)
	default: // Could not determine how file upload was initiated, grabbing the request body
		rb, err = c.readCloserFromRawBody(r)
	}
	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), err.(*httpError).stat)
		events.Raise(ctx, events.ClientDisconnected{Err: err})
		return
	}

	src := rb.r
	if c.decodeBase64Output && 0 < rb.size {
		src = io.NopCloser(base64.NewDecoder(base64.StdEncoding, src))
	}

	fileSize := int(rb.size)
	if fileSize != 0 {
		// if decoding base64
		if c.decodeBase64Output {
			fileSize = base64.StdEncoding.DecodedLen(fileSize)
		}
	}

	wts, err := c.fileTransferConfig.NewWriteTransferSession(ctx, rb.name, rb.mime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		events.Raise(ctx, events.ClientDisconnected{Err: err})
		return
	}
	defer wts.Close()

	cancelProgDisp := output.DisplayProgress(
		ctx,
		&wts.Progress,
		125*time.Millisecond,
		r.RemoteAddr,
		int64(fileSize),
	)
	defer cancelProgDisp()

	file, getBufBytes := output.NewBufferedWriter(ctx, wts)
	fileReport := events.File{
		MIME:              rb.mime,
		Size:              int64(fileSize),
		Name:              rb.name,
		TransferStartTime: time.Now(),
	}
	fileReport.TransferSize, err = io.Copy(file, src)
	fileReport.TransferEndTime = time.Now()
	if err != nil {
		events.Raise(ctx, &fileReport)
		events.Raise(ctx, events.ClientDisconnected{
			Err: err,
		})
		return
	}

	w.WriteHeader(c.statusCode)

	fileReport.Path = wts.Path()
	fileReport.Content = getBufBytes
	events.Raise(ctx, &fileReport)

	events.Success(ctx)
}

func (c *Cmd) _handleGET(w http.ResponseWriter, r *http.Request) {
	w.(oneshothttp.ResponseWriter).IgnoreOutcome()

	withJS := true
	ua := r.Header.Get("User-Agent")
	if strings.Contains(ua, "curl") || strings.Contains(ua, "wget") {
		withJS = false
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.writeTemplate(w, withJS); err != nil {
		log.Printf("error writing template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
