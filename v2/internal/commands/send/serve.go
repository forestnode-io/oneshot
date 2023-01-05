package send

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/v2/internal/api"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
	"github.com/raphaelreyna/oneshot/v2/internal/out/events"
)

func (s *Cmd) ServeHTTP(actx api.Context, w http.ResponseWriter, r *http.Request) {
	var (
		file = s.file

		cmd           = s.cmd
		flags         = cmd.Flags()
		noDownload, _ = flags.GetBool("no-download")
		status, _     = flags.GetInt("status-code")

		header = s.header
	)

	actx.Raise(out.NewHTTPRequest(r))

	err := file.Open()
	defer func() {
		file.Reset()
	}()
	if err := r.Context().Err(); err != nil {
		actx.Raise(&events.ClientDisconnected{
			Err: err,
		})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		actx.Raise(&events.ClientDisconnected{
			Err: err,
		})
		return
	}

	// Are we triggering a file download on the users browser?
	if !noDownload {
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment;filename=%s", file.Name),
		)
	}

	// Set standard Content headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", file.GetSize()))

	// Set any additional headers added by the user via flags
	for key := range header {
		w.Header().Set(key, header.Get(key))
	}
	w.WriteHeader(status)

	eventFile := &events.File{
		Name: file.Name,
		MIME: file.MimeType,
		Size: file.GetSize(),
	}

	success := false
	cancelProgDisp := out.DisplayProgress(
		&file.Progress,
		125*time.Millisecond,
		r.RemoteAddr,
		file.GetSize(),
	)
	defer func() {
		cancelProgDisp(success)
	}()

	// Start writing the file data to the client while timing how long it takes
	n, err := io.Copy(w, file)
	writeSize := n
	if err != nil {
		actx.Raise(events.ClientDisconnected{
			Err: err,
		})
		return
	}

	if writeSize != eventFile.Size {
		actx.Raise(&events.ClientDisconnected{
			Err: err,
		})
		return
	}

	actx.Raise(eventFile)

	actx.Success()
	success = true
}

func (d *Cmd) ServeExpiredHTTP(_ api.Context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
