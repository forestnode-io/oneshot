package send

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/v2/internal/api"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
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
		actx.Raise(&out.ClientDisconnected{
			Err: err,
		})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		actx.Raise(&out.ClientDisconnected{
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
	w.Header().Set("Content-Length", fmt.Sprintf("%d", file.Size()))

	// Set any additional headers added by the user via flags
	for key := range header {
		w.Header().Set(key, header.Get(key))
	}
	w.WriteHeader(status)

	eventFile := &out.File{
		Name: file.Name,
		MIME: file.MimeType,
		Size: file.Size(),
	}

	rp, wp := io.Pipe()
	defer rp.Close()
	defer wp.Close()
	file.ProgressWriter = wp
	actx.Raise(out.TransferProgress(func(w io.Writer) *out.TransferInfo {
		ti := out.TransferInfo{
			WriteStartTime: time.Now(),
		}

		n, _ := io.Copy(w, rp)
		w.Write([]byte("\n"))

		ti.WriteEndTime = time.Now()
		ti.WriteDuration = ti.WriteEndTime.Sub(ti.WriteStartTime)
		ti.WriteSize = n
		ti.WriteBytesPerSecond = 1000 * 1000 * 1000 * n / int64(ti.WriteDuration)

		return &ti
	}))

	// Start writing the file data to the client while timing how long it takes
	n, err := io.Copy(w, file)
	writeSize := n
	if err != nil {
		actx.Raise(out.ClientDisconnected{
			Err: err,
		})
		return
	}

	if writeSize != eventFile.Size {
		actx.Raise(&out.ClientDisconnected{
			Err: err,
		})
		return
	}

	actx.Raise(eventFile)

	actx.Success()
}

func (d *Cmd) ServeExpiredHTTP(_ api.Context, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
