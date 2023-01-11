package send

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
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

	events.Raise(ctx, output.NewHTTPRequest(r))

	if err := file.Open(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		output.ClientDisconnected(ctx, err)
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

	cancelProgDisp := output.DisplayProgress(
		cmd.Context(),
		&file.Progress,
		125*time.Millisecond,
		r.RemoteAddr,
		file.GetSize(),
	)
	defer cancelProgDisp()

	// Start writing the file data to the client while timing how long it takes
	n, err := io.Copy(w, file)
	writeSize := n
	if err != nil {
		output.ClientDisconnected(ctx, err)
		return
	}

	if writeSize != eventFile.Size {
		output.ClientDisconnected(ctx, err)
		return
	}

	output.Raise(ctx, eventFile)
	events.Success(ctx)
}

func (d *Cmd) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
