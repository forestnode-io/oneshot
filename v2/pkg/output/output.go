package output

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/muesli/termenv"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
)

type key struct{}

func getOutput(ctx context.Context) *output {
	o, _ := ctx.Value(key{}).(*output)
	if o == nil {
		panic("no output set")
	}
	return o
}

type output struct {
	events chan events.Event

	stdoutTTY *termenv.Output
	stderrTTY *termenv.Output

	dynamicOutput *tabbedDynamicOutput

	Format     string
	FormatOpts []string

	skipSummary     bool
	servingToStdout bool
	receivedBuf     *bytes.Buffer

	cls                  []*clientSession
	currentClientSession *clientSession

	quiet bool

	doneChan chan struct{}

	stdinIsTTY bool

	displayProgresssPeriod    time.Duration
	lastProgressDisplayAmount int64

	restoreConsole  []func() error
	stdoutFailColor termenv.Color
	stderrFailColor termenv.Color
}

func (o *output) run(ctx context.Context) error {
	if o.quiet {
		runQuiet(ctx, o)
	} else {
		switch o.Format {
		case "":
			runHuman(ctx, o)
		case "json":
			NewHTTPRequest = events.NewHTTPRequest_WithBody
			runJSON(ctx, o)
		}

		// is outputting human readable content to stdout
		if o.servingToStdout && o.Format != "json" {
			// add a newline.
			// some shells will add a EOF character otherwise
			fmt.Fprint(os.Stdout, "\n")
		}
	}

	for _, f := range o.restoreConsole {
		f()
	}

	o.doneChan <- struct{}{}
	return nil
}

func (o *output) writeListeningOnQRCode(scheme, host, port string) {
	qrConf := qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     os.Stderr,
		BlackChar:  qrterminal.BLACK,
		WhiteChar:  qrterminal.WHITE,
		QuietZone:  1,
		HalfBlocks: false,
	}
	if o.Format == "json" || o.skipSummary {
		return
	}

	if host == "" {
		addrs, err := oneshotnet.HostAddresses()
		if err != nil {
			addr := fmt.Sprintf("%s://localhost%s", scheme, port)
			fmt.Fprintf(os.Stderr, "%s:\n", addr)
			qrterminal.GenerateWithConfig(addr, qrConf)
			return
		}

		fmt.Fprintln(os.Stderr, "listening on: ")
		for _, addr := range addrs {
			addr = fmt.Sprintf("%s://%s", scheme, oneshotfmt.Address(addr, port))
			fmt.Fprintf(os.Stderr, "%s:\n", addr)
			qrterminal.GenerateWithConfig(addr, qrConf)
		}
		return
	}

	addr := fmt.Sprintf("%s://%s", scheme, oneshotfmt.Address(host, port))
	fmt.Fprintf(os.Stderr, "%s:\n", addr)
	qrterminal.GenerateWithConfig(addr, qrConf)
}

// ttyCheck checks if stdin, stdout, and stderr are ttys
// and records it in o.
func (o *output) ttyCheck() error {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return err
	}
	stdoutIsTTY := (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	//o.tabbedStdout = tabwriter.NewWriter(o.Stdout, 12, 2, 2, ' ', 0)

	stat, err = os.Stderr.Stat()
	if err != nil {
		return err
	}
	stderrIsTTY := (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice

	stat, err = os.Stderr.Stat()
	if err != nil {
		return err
	}
	o.stdinIsTTY = (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice

	if os.Getenv("ONESHOT_TESTING_TTY_STDOUT") != "" {
		stdoutIsTTY = true
	}

	if os.Getenv("ONESHOT_TESTING_TTY_STDERR") != "" {
		stderrIsTTY = true
	}

	if os.Getenv("ONESHOT_TESTING_TTY_STDIN") != "" {
		o.stdinIsTTY = true
	}

	if stdoutIsTTY {
		o.stdoutTTY = termenv.DefaultOutput()
		restoreStdout, err := termenv.EnableVirtualTerminalProcessing(o.stdoutTTY)
		if err != nil {
			return err
		}
		if !o.stdoutTTY.EnvNoColor() {
			o.stdoutFailColor = o.stdoutTTY.Color("#ff0000")
		}
		o.restoreConsole = append(o.restoreConsole, restoreStdout)
	}

	if stderrIsTTY {
		o.stderrTTY = termenv.NewOutput(os.Stderr)
		restoreStderr, err := termenv.EnableVirtualTerminalProcessing(o.stderrTTY)
		if err != nil {
			return err
		}
		if !o.stderrTTY.EnvNoColor() {
			o.stderrFailColor = o.stderrTTY.Color("#ff0000")
		}
		o.restoreConsole = append(o.restoreConsole, restoreStderr)
	}

	return nil
}

func (o *output) enableDynamicOutput(te *termenv.Output) {
	if te == nil {
		if o.stdoutTTY != nil {
			te = o.stdoutTTY
		} else if o.stderrTTY != nil {
			te = o.stderrTTY
		}
	}

	if te != nil {
		o.dynamicOutput = newTabbedDynamicOutput(te)
	}

	if o.dynamicOutput == nil {
		return
	}

	o.dynamicOutput.HideCursor()
	o.restoreConsole = append(o.restoreConsole, func() error {
		if o.dynamicOutput != nil {
			o.dynamicOutput.ShowCursor()
			o.dynamicOutput = nil
		}
		return nil
	})
}

type clientSession struct {
	Request *events.HTTPRequest `json:",omitempty"`
	File    *events.File        `json:",omitempty"`
}

type report struct {
	Success  *clientSession
	Attempts []*clientSession
}

func bytesPerSecond(bytes int64, dt time.Duration) float64 {
	const floatSecond = float64(time.Second)
	return float64(bytes) / float64(dt) * floatSecond
}
