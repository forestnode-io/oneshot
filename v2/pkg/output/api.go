package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"
	"text/tabwriter"
	"time"

	"github.com/muesli/termenv"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
)

func WithOutput(ctx context.Context) (context.Context, error) {
	o := output{
		doneChan:   make(chan struct{}),
		cls:        make([]*clientSession, 0),
		FormatOpts: map[string]struct{}{},
	}

	if err := o.ttyCheck(); err != nil {
		return nil, err
	}

	return context.WithValue(ctx, key{}, &o), nil
}

func InvocationInfo(ctx context.Context, cmdName string, argc int) {
	o := getOutput(ctx)
	switch cmdName {
	case "exec":
		log.SetPrefix("oneshot exec: ")
		if o.Format == "json" {
			// if outputting json report and executing a command, include the sent body in the report
			// since the user may not have a copy of it laying around
			if _, exclude := o.FormatOpts["exclude-file-contents"]; !exclude {
				o.FormatOpts["include-file-contents"] = struct{}{}
			}
		}
	case "redirect":
		log.SetPrefix("oneshot redirect: ")
	case "send":
		log.SetPrefix("oneshot send: ")
		switch argc {
		case 0: // sending from stdin
			if o.Format != "json" {
				// if stdin is not a tty we can try dynamic output to the tty
				if !o.stdinIsTTY {
					o.enableDynamicOutput(nil)
				} else {
					o.ttyForContentOnly = true
				}
			} else {
				// if outputting json report and sending from stdin, include the sent body in the report
				// since the user may not have a copy of it laying around
				if _, exclude := o.FormatOpts["exclude-file-contents"]; !exclude {
					o.FormatOpts["include-file-contents"] = struct{}{}
				}
			}
		default: // sending file(s)
			if o.Format != "json" {
				o.enableDynamicOutput(nil)
			}
		}
	case "receive":
		log.SetPrefix("oneshot receive: ")
		switch argc {
		case 0: // receiving to stdout
			if o.Format != "json" {
				if o.stdoutTTY != nil {
					o.ttyForContentOnly = true
				}

				// try to fallback to stderr for dynamic out output but only if
				// stdout is not a tty since the stderr tty is usually the same as the stdout tty.
				if o.dynamicOutput != nil {
					o.dynamicOutput = nil
					if o.stdoutTTY == nil && o.stderrTTY != nil {
						o.enableDynamicOutput(o.stderrTTY)
					}
				}
			} else {
				// if outputting json report and receiving to stdout, include the received body in the report
				// since the user may not have a copy of it laying around
				if _, exclude := o.FormatOpts["exclude-file-contents"]; !exclude {
					o.FormatOpts["include-file-contents"] = struct{}{}
				}
			}
		default: // receiving to file
			if o.Format != "json" {
				o.enableDynamicOutput(nil)
			}
		}
	default:
	}
}

// TODO(raphaelreyna): move this into the events package
func ClientDisconnected(ctx context.Context, err error) {
	getOutput(ctx).events <- &events.ClientDisconnected{
		Err: err,
	}
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
	o := getOutput(ctx)
	for _, opt := range opts {
		o.FormatOpts[opt] = struct{}{}
	}
}

func GetFormatAndOpts(ctx context.Context) (string, map[string]struct{}) {
	o := getOutput(ctx)
	return o.Format, o.FormatOpts
}

func IsServingToStdout(ctx context.Context) bool {
	return getOutput(ctx).ttyForContentOnly
}

func NoColor(ctx context.Context) {
	o := getOutput(ctx)
	o.stdoutFailColor = nil
	o.stderrFailColor = nil
}

// ReceivingToStdout ensures that only the appropriate content is sent to stdout.
// The summary is flagged to be skipped and if outputting json, make sure we have initiated the buffer
// that holds the received content.
func ReceivingToStdout(ctx context.Context) {
	o := getOutput(ctx)

	o.skipSummary = true
	o.ttyForContentOnly = true
	if o.Format == "json" {
		if o.receivedBuf == nil {
			o.receivedBuf = bytes.NewBuffer(nil)
		}
	}
}

func Init(ctx context.Context) {
	go func() {
		if err := getOutput(ctx).run(ctx); err != nil {
			log.Printf("error running output system: %v", err)
		}
	}()
}

func Wait(ctx context.Context) {
	<-getOutput(ctx).doneChan
}

func GetBufferedWriteCloser(ctx context.Context) io.WriteCloser {
	o := getOutput(ctx)
	return &writer{os.Stdout, o.receivedBuf}
}

func DisplayProgress(ctx context.Context, prog *atomic.Int64, period time.Duration, host string, total int64) func() {
	o := getOutput(ctx)
	if o.quiet || o.Format == "json" {
		return func() {}
	}

	var (
		done chan struct{}

		start  = time.Now()
		prefix = fmt.Sprintf("%s\t%s", start.Format(progDisplayTimeFormat), host)
	)

	if o.dynamicOutput != nil {
		o.displayProgresssPeriod = period
		displayDynamicProgress(o, prefix, start, prog, total)

		done = make(chan struct{})
		ticker := time.NewTicker(period)

		go func() {
			for {
				select {
				case <-done:
					ticker.Stop()
					return
				case <-ticker.C:
					displayDynamicProgress(o, prefix, start, prog, total)
				}
			}
		}()
	}

	return func() {
		if done != nil {
			done <- struct{}{}
			close(done)
			done = nil
		}

		if events.Succeeded(ctx) {
			displayProgressSuccessFlush(o, prefix, start, prog.Load())
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

type tabbedDynamicOutput struct {
	tw *tabwriter.Writer
	te *termenv.Output
}

func newTabbedDynamicOutput(te *termenv.Output) *tabbedDynamicOutput {
	return &tabbedDynamicOutput{
		tw: tabwriter.NewWriter(te, 12, 2, 2, ' ', 0),
		te: termenv.NewOutput(te),
	}
}

func (o *tabbedDynamicOutput) resetLine() {
	_, err := o.te.WriteString("\r")
	if err != nil {
		log.Printf("error writing carriage-return character: %v", err)
	}
	o.te.ClearLineRight()
}

func (o *tabbedDynamicOutput) flush() error {
	return o.tw.Flush()
}

func (o *tabbedDynamicOutput) Write(p []byte) (int, error) {
	return o.tw.Write(p)
}

func (o *tabbedDynamicOutput) ShowCursor() {
	o.te.ShowCursor()
}

func (o *tabbedDynamicOutput) HideCursor() {
	o.te.HideCursor()
}

func (o *tabbedDynamicOutput) EnvNoColor() bool {
	return o.te.EnvNoColor()
}

func (o *tabbedDynamicOutput) Color(s string) termenv.Color {
	return o.te.Color(s)
}

func (o *tabbedDynamicOutput) String(s string) termenv.Style {
	return o.te.String(s)
}
