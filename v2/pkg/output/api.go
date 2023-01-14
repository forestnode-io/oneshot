package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"text/tabwriter"
	"time"

	"github.com/muesli/termenv"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
)

func WithOutput(ctx context.Context) (context.Context, error) {
	o := output{
		Stdout:   termenv.DefaultOutput(),
		Stderr:   termenv.NewOutput(os.Stderr),
		doneChan: make(chan struct{}),
		cls:      make([]*clientSession, 0),
	}

	restoreStdout, err := termenv.EnableVirtualTerminalProcessing(o.Stdout)
	if err != nil {
		return nil, err
	}
	if !o.Stdout.EnvNoColor() {
		o.stdoutFailColor = o.Stdout.Color("#ff0000")
	}

	restoreStderr, err := termenv.EnableVirtualTerminalProcessing(o.Stderr)
	if err != nil {
		return nil, err
	}
	if !o.Stderr.EnvNoColor() {
		o.stderrFailColor = o.Stderr.Color("#ff0000")
	}

	o.restoreConsole = func() {
		restoreStdout()
		restoreStderr()
	}

	stat, err := os.Stdout.Stat()
	if err != nil {
		return nil, err
	}
	o.stdoutIsTTY = (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	o.tabbedStdout = tabwriter.NewWriter(o.Stdout, 12, 2, 2, ' ', 0)

	stat, err = os.Stderr.Stat()
	if err != nil {
		return nil, err
	}
	o.stderrIsTTY = (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	o.tabbedStderr = tabwriter.NewWriter(o.Stderr, 12, 2, 2, ' ', 0)

	return context.WithValue(ctx, key{}, &o), nil
}

func ClientDisconnected(ctx context.Context, err error) {
	Raise(ctx, &events.ClientDisconnected{
		Err: err,
	})
}

func Raise(ctx context.Context, e events.Event) {
	getOutput(ctx).events <- e
}

func SetEventsChan(ctx context.Context, ec chan events.Event) {
	getOutput(ctx).events = ec
}

func WriteListeningOnQR(ctx context.Context, scheme, host, port string) {
	getOutput(ctx).writeListeningOnQRCode(scheme, host, port)
}

func Quiet(ctx context.Context) {
	getOutput(ctx).quiet = true
}

func SetFormat(ctx context.Context, f string) {
	getOutput(ctx).Format = f
}

func SetFormatOpts(ctx context.Context, opts ...string) {
	getOutput(ctx).FormatOpts = opts
}

func GetFormatAndOpts(ctx context.Context) (string, []string) {
	o := getOutput(ctx)
	return o.Format, o.FormatOpts
}

func IsServingToStdout(ctx context.Context) bool {
	return getOutput(ctx).servingToStdout
}

// ReceivingToStdout ensures that only the appropriate content is sent to stdout.
// The summary is flagged to be skipped and if outputting json, make sure we have initiated the buffer
// that holds the received content.
func ReceivingToStdout(ctx context.Context) {
	o := getOutput(ctx)

	o.skipSummary = true
	o.servingToStdout = true
	if o.Format == "json" {
		if o.receivedBuf == nil {
			o.receivedBuf = bytes.NewBuffer(nil)
		}
	}
}

func Init(ctx context.Context) error {
	go getOutput(ctx).run(ctx)
	return nil
}

func Wait(ctx context.Context) {
	<-getOutput(ctx).doneChan
}

func GetWriteCloser(ctx context.Context) io.WriteCloser {
	o := getOutput(ctx)
	return &writer{o.Stdout, o.receivedBuf}
}

func DisplayProgress(ctx context.Context, prog *atomic.Int64, period time.Duration, host string, total int64) func() {
	o := getOutput(ctx)
	if o.servingToStdout || o.Format == "json" || o.quiet {
		return func() {}
	}

	var (
		start  = time.Now()
		prefix = fmt.Sprintf("%s\t%s", start.Format(progDisplayTimeFormat), host)
	)
	if !o.stderrIsTTY {
		return func() {
			if events.Succeeded(ctx) {
				displayProgressSuccessFlush(o, prefix, start, prog.Load())
			} else {
				displayProgressFailFlush(o, prefix, start, prog.Load(), total)
			}
		}
	}

	var (
		ticker = time.NewTicker(period)
		done   = make(chan struct{})

		lastTime = start
	)

	o.displayProgresssPeriod = period
	lastTime = displayProgress(o, prefix, start, prog, total)

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				lastTime = displayProgress(o, prefix, start, prog, total)
			}
		}
	}()

	return func() {
		if done == nil {
			return
		}

		done <- struct{}{}
		close(done)
		done = nil

		if events.Succeeded(ctx) {
			displayProgressSuccessFlush(o, prefix, lastTime, prog.Load())
		} else {
			displayProgressFailFlush(o, prefix, start, prog.Load(), total)
		}
	}
}

func NewBufferedWriter(ctx context.Context, w io.Writer) (io.Writer, func() []byte) {
	o := getOutput(ctx)
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
