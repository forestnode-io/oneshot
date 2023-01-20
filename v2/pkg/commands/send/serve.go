package send

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
)

func (c *Cmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = c.Cobra().Context()

		cmd = c.cobraCommand

		header = c.header
	)

	rts, err := c.rtc.NewReaderTransferSession(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		output.ClientDisconnected(ctx, err)
		return
	}
	defer rts.Close()
	size, err := rts.Size()
	if err == nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	}

	for key := range header {
		w.Header().Set(key, header.Get(key))
	}
	w.WriteHeader(c.status)

	cancelProgDisp := output.DisplayProgress(
		cmd.Context(),
		&rts.Progress,
		125*time.Millisecond,
		r.RemoteAddr,
		0,
	)
	defer cancelProgDisp()

	// Start writing the file data to the client while timing how long it takes
	if _, err = io.Copy(w, rts); err != nil {
		output.ClientDisconnected(ctx, err)
		return
	}

	events.Success(ctx)
}

func (d *Cmd) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
