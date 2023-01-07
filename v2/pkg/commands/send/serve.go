package send

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/out"
)

func (s *Cmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = s.Cobra().Context()
		file = s.file

		cmd           = s.cobraCommand
		flags         = cmd.Flags()
		noDownload, _ = flags.GetBool("no-download")
		status, _     = flags.GetInt("status-code")

		header = s.header
	)

	events.Raise(ctx, out.NewHTTPRequest(r))

	if err := file.Open(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		out.ClientDisconnected(ctx, err)
		return
	}
	defer func() {
		file.Reset()
	}()

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
		cmd.Context(),
		&file.Progress,
		125*time.Millisecond,
		r.RemoteAddr,
		file.GetSize(),
	)
	defer func() {
		cancelProgDisp(success)
		if success {
			events.Success(ctx)
		}
	}()

	// Start writing the file data to the client while timing how long it takes
	n, err := io.Copy(w, file)
	writeSize := n
	if err != nil {
		out.ClientDisconnected(ctx, err)
		return
	}

	if writeSize != eventFile.Size {
		out.ClientDisconnected(ctx, err)
		return
	}

	out.Raise(ctx, eventFile)
	success = true
}

func (d *Cmd) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
