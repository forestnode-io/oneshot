package receive

import (
	"io"
	"net/http"
	"strings"

	"github.com/raphaelreyna/oneshot/v2/internal/api"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
)

func (c *Cmd) ServeHTTP(actx api.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		c._handleGET(w, r)
		return
	}

	actx.Raise(out.NewHTTPRequest(r))

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
		src, cl, err = c.readerFromMultipartFormData(r)
	case strings.Contains(rct, "application/x-www-form-urlencoded"): // User uploaded text from HTML text box
		src, cl, err = c.readerFromApplicationWWWForm(r)
	default: // Could not determine how file upload was initiated, grabbing the request body
		src, cl, err = c.readerFromRawBody(r)
	}

	if err != nil {
		http.Error(w, err.Error(), err.(*httpError).stat)
		actx.Raise(&out.ClientDisconnected{
			Err: err,
		})
		return
	}

	c.file.Lock()
	defer c.file.Unlock()
	defer r.Body.Close()

	if cl != 0 {
		c.file.SetSize(cl)
	}

	// open the file we are writing to
	if err = c.file.Open(c.cobraCommand.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		actx.Raise(&out.ClientDisconnected{
			Err: err,
		})
		return
	}

	_, err = io.Copy(c.file, src)
	if err != nil {
		c.file.Reset()
		actx.Raise(&out.ClientDisconnected{
			Err: err,
		})
		return
	}
	c.file.Close()

	actx.Raise(&out.File{
		MIME: c.file.MIMEType,
		Size: c.file.GetSize(),
		Path: c.file.GetLocation(),
		Name: c.file.Name(),
	})
	actx.Success()
}

func (c *Cmd) _handleGET(w http.ResponseWriter, r *http.Request) {
	c.writeTemplate(w)
}

func (c *Cmd) ServeExpiredHTTP(_ api.Context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
