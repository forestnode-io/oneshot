package out

import (
	"bytes"
	"io"
	"time"
)

func WriteListeningOn(scheme, host, port string) {
	o.writeListeningOn(scheme, host, port)
}

func SetEventsChan(ec <-chan Event) {
	o.Events = ec
}

func SetStdout(w io.Writer) {
	o.Stdout = w
}

func SetFormat(f string) {
	o.Format = f
}

func SetFormatOpts(opts ...string) {
	o.FormatOpts = opts
}

func GetFormatAndOpts() (string, []string) {
	return o.Format, o.FormatOpts
}

func IsServingToStdout() bool {
	return o.servingToStdout
}

// ReceivingToStdout ensures that only the appropriate content is sent to stdout.
// The summary is flagged to be skipped and if outputting json, make sure we have initiated the buffer
// that holds the received content.
func ReceivingToStdout() {
	o.skipSummary = true
	o.servingToStdout = true
	if o.Format == "json" {
		if o.receivedBuf == nil {
			o.receivedBuf = bytes.NewBuffer(nil)
		}
	}
}

func Init() {
	go o.run()
}

func Wait() {
	<-o.doneChan
}

func NewProgressWriter() (io.WriteCloser, Event, func()) {
	if o.Format == "json" {
		return nil, nopTransferProgress, func() {}
	}

	rp, wp := io.Pipe()
	var tpFunc TransferProgress = func(w io.Writer) *transferInfo {
		ti := transferInfo{
			WriteStartTime: time.Now(),
		}

		n, _ := io.Copy(w, rp)
		w.Write([]byte("\n"))

		ti.WriteEndTime = time.Now()
		ti.WriteDuration = ti.WriteEndTime.Sub(ti.WriteStartTime)
		ti.WriteSize = n
		ti.WriteBytesPerSecond = 1000 * 1000 * 1000 * n / int64(ti.WriteDuration)

		return &ti
	}

	return wp, tpFunc, func() {
		rp.Close()
		wp.Close()
	}
}

func GetWriteCloser() io.WriteCloser {
	return &writer{o.Stdout, o.receivedBuf}
}

func NewBufferedWriter(w io.Writer) (io.Writer, func() []byte) {
	if o.Format != "json" {
		return w, nil
	}

	buf := bytes.NewBuffer(nil)
	tw := teeWriter{
		w:    w,
		copy: buf,
	}

	return tw, buf.Bytes
}

type teeWriter struct {
	w, copy io.Writer
}

func (t teeWriter) Write(p []byte) (n int, err error) {
	n, err = t.w.Write(p)
	if n > 0 {
		if n, err := t.copy.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return

}

type writer struct {
	w   io.Writer
	buf *bytes.Buffer
}

func (w *writer) Write(p []byte) (int, error) {
	if b := w.buf; b != nil {
		return b.Write(p)
	}
	return w.w.Write(p)
}

func (*writer) Close() error {
	return nil
}
