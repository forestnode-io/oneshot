package out

import (
	"bytes"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/raphaelreyna/oneshot/v2/internal/out/events"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/internal/out/fmt"
)

func WriteListeningOnQR(scheme, host, port string) {
	o.writeListeningOnQRCode(scheme, host, port)
}

func SetEventsChan(ec <-chan events.Event) {
	o.Events = ec
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

func DisplayProgress(prog *atomic.Int64, period time.Duration, host string, total int64) func(bool) {
	if o.servingToStdout || o.Format == "json" {
		return func(_ bool) {}
	}

	var (
		start  = time.Now()
		ticker = time.NewTicker(period)
		done   = make(chan struct{})

		lastTime = start
	)

	fmt.Fprintf(o.Stdout, "%s  %s", start.Format(progDisplayTimeFormat), host)
	o.Stdout.SaveCursorPosition()

	lastTime = displayProgress(start, prog, total)

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				lastTime = displayProgress(start, prog, total)
			}
		}
	}()

	return func(success bool) {
		if done == nil {
			return
		}

		done <- struct{}{}
		close(done)
		done = nil

		if success {
			displayProgressSuccessFlush(lastTime, prog.Load())
		}
	}
}

const progDisplayTimeFormat = "2006-01-02T15:04:05-0700"

func displayProgressSuccessFlush(start time.Time, total int64) {
	const (
		kb = 1000
		mb = kb * 1000
		gb = mb * 1000
	)

	o.Stdout.ClearLineRight()
	o.Stdout.RestoreCursorPosition()

	switch {
	case total < kb:
		fmt.Fprintf(o.Stdout, "%8d B  ", total)
	case total < mb:
		fmt.Fprintf(o.Stdout, "%8.2f KB  ", float64(total)/kb)
	case total < gb:
		fmt.Fprintf(o.Stdout, "%8.2f MB  ", float64(total)/mb)
	default:
		fmt.Fprintf(o.Stdout, "%8.2f GB  ", float64(total)/gb)
	}

	duration := time.Since(start)
	rate := 1000 * 1000 * 1000 * total / int64(duration)
	fmt.Fprintf(o.Stdout, "%v  100%%  0s  %v  ...success\n",
		oneshotfmt.PrettyRate(rate),
		oneshotfmt.RoundedDurationString(duration, 2),
	)
}

func displayProgress(start time.Time, prog *atomic.Int64, total int64) time.Time {
	const (
		kb = 1000
		mb = kb * 1000
		gb = mb * 1000
	)

	var progress = prog.Load()

	o.Stdout.ClearLineRight()
	o.Stdout.RestoreCursorPosition()

	switch {
	case progress < kb:
		fmt.Fprintf(o.Stdout, "%8d B  ", progress)
	case progress < mb:
		fmt.Fprintf(o.Stdout, "%8.2f KB  ", float64(progress)/kb)
	case progress < gb:
		fmt.Fprintf(o.Stdout, "%8.2f MB  ", float64(progress)/mb)
	default:
		fmt.Fprintf(o.Stdout, "%8.2f GB  ", float64(progress)/gb)
	}

	duration := time.Since(start)
	rate := 1000 * 1000 * 1000 * progress / int64(duration)
	fmt.Fprintf(o.Stdout, "%v  ", oneshotfmt.PrettyRate(rate))
	if total != 0 {
		percent := 100.0 * float64(progress) / float64(total)
		if rate != 0 {
			timeLeft := (total - progress) / rate
			fmt.Fprintf(o.Stdout, "%8.2f%%  %d  ", percent, timeLeft)
		} else {
			fmt.Fprintf(o.Stdout, "%8.2f%%  n/a  ", percent)
		}
	} else {
		fmt.Fprintf(o.Stdout, "n/a  n/a  ")
	}
	fmt.Fprintf(o.Stdout, "%v  ", oneshotfmt.RoundedDurationString(duration, 2))

	return start
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
