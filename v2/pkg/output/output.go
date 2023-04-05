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
	"github.com/rs/zerolog"
)

var NewHTTPRequest = events.NewHTTPRequest

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
	FormatOpts map[string]struct{}

	skipSummary       bool
	ttyForContentOnly bool
	includeBody       bool
	receivedBuf       *bytes.Buffer

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

	cmdName string
}

func (o *output) run(ctx context.Context) error {
	log := zerolog.Ctx(ctx)

	if o.quiet {
		log.Debug().
			Msg("output running in quiet mode")
		runQuiet(ctx, o)
	} else {
		switch o.Format {
		case "":
			log.Debug().
				Msg("output running in human mode")
			runHuman(ctx, o)
		case "json":
			log.Debug().
				Msg("output running in json mode")
			NewHTTPRequest = events.NewHTTPRequest_WithBody
			runJSON(ctx, o)
		}

		// is outputting human readable content to stdout
		if o.ttyForContentOnly && o.Format != "json" && events.Succeeded(ctx) {
			// add a newline.
			// some shells will add a EOF character otherwise
			fmt.Fprint(os.Stdout, "\n")
		}
	}

	log.Debug().
		Msg("output system shutting down")

	for _, f := range o.restoreConsole {
		if err := f(); err != nil {
			log.Error().Err(err).
				Msg("error from console restore func")
		}
	}

	close(o.doneChan)

	log.Debug().
		Msg("output system shut down")

	return nil
}

func (o *output) writeListeningOnQRCode(addr string) {
	if o.Format == "json" || o.skipSummary || o.quiet || addr == "" {
		return
	}

	qrConf := qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     os.Stderr,
		BlackChar:  qrterminal.BLACK,
		WhiteChar:  qrterminal.WHITE,
		QuietZone:  1,
		HalfBlocks: false,
	}

	fmt.Fprintln(os.Stderr, addr)
	qrterminal.GenerateWithConfig(addr, qrConf)
}

func (o *output) writeListeningOn(addr string) {
	if o.Format == "json" || o.skipSummary || o.quiet || addr == "" {
		return
	}

	fmt.Fprintf(os.Stderr, "listening on %s\n", addr)
}

// ttyCheck checks if stdin, stdout, and stderr are ttys
// and records it in o.
func (o *output) ttyCheck() error {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return err
	}
	stdoutIsTTY := (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice

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

	// if we're running in a docker container, assume both stdout and stderr are ttys
	if _, err := os.Stat("/.dockerenv"); err == nil {
		stdoutIsTTY = true
		stderrIsTTY = true
	}

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
	Request  *events.HTTPRequest  `json:",omitempty"`
	File     *events.File         `json:",omitempty"`
	Response *events.HTTPResponse `json:",omitempty"`
	Error    string               `json:",omitempty"`
}

type Report struct {
	Success  *clientSession   `json:",omitempty"`
	Attempts []*clientSession `json:",omitempty"`
}

func bytesPerSecond(bytes int64, dt time.Duration) float64 {
	const floatSecond = float64(time.Second)
	return float64(bytes) / float64(dt) * floatSecond
}
