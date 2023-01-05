package out

import (
	"bytes"
	"fmt"

	"github.com/mdp/qrterminal/v3"
	"github.com/muesli/termenv"
	"github.com/raphaelreyna/oneshot/v2/internal/network"
	"github.com/raphaelreyna/oneshot/v2/internal/out/events"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/internal/out/fmt"
)

type out struct {
	Events     <-chan events.Event
	Stdout     *termenv.Output
	Format     string
	FormatOpts []string

	skipSummary     bool
	servingToStdout bool
	receivedBuf     *bytes.Buffer

	cls                  []*clientSession
	currentClientSession *clientSession

	doneChan chan struct{}
}

var o = out{
	Stdout:   termenv.DefaultOutput(),
	doneChan: make(chan struct{}),
	cls:      make([]*clientSession, 0),
}

func (o *out) run() {
	if !o.servingToStdout {
		o.Stdout.HideCursor()
		defer o.Stdout.ShowCursor()
	}

	switch o.Format {
	case "":
		o.runHuman()
	case "json":
		NewHTTPRequest = events.NewHTTPRequest_WithBody
		o.runJSON()
	}

	o.doneChan <- struct{}{}
}

func (o *out) writeListeningOnQRCode(scheme, host, port string) {
	qrConf := qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     o.Stdout,
		BlackChar:  qrterminal.BLACK,
		WhiteChar:  qrterminal.WHITE,
		QuietZone:  1,
		HalfBlocks: false,
	}
	if o.Format == "json" || o.skipSummary {
		return
	}

	if host == "" {
		addrs, err := network.HostAddresses()
		if err != nil {
			addr := fmt.Sprintf("%s://localhost%s", scheme, port)
			fmt.Fprintf(o.Stdout, "%s:\n", addr)
			qrterminal.GenerateWithConfig(addr, qrConf)
			return
		}

		fmt.Fprintln(o.Stdout, "listening on: ")
		for _, addr := range addrs {
			addr = fmt.Sprintf("%s://%s", scheme, oneshotfmt.Address(addr, port))
			fmt.Fprintf(o.Stdout, "%s:\n", addr)
			qrterminal.GenerateWithConfig(addr, qrConf)
		}
		return
	}

	addr := fmt.Sprintf("%s://%s", scheme, oneshotfmt.Address(host, port))
	fmt.Fprintf(o.Stdout, "%s:\n", addr)
	qrterminal.GenerateWithConfig(addr, qrConf)
}

func (o *out) writeListeningOn(scheme, host, port string) {
	if o.Format == "json" || o.skipSummary {
		return
	}

	if host == "" {
		addrs, err := network.HostAddresses()
		if err != nil {
			fmt.Fprintf(o.Stdout, "Listening on: %s://localhost%s\n", scheme, port)
			return
		}

		fmt.Fprintln(o.Stdout, "Listening on: ")
		for _, addr := range addrs {
			fmt.Fprintf(o.Stdout, "  - %s://%s\n", scheme, oneshotfmt.Address(addr, port))
		}
		return
	}

	fmt.Fprintf(o.Stdout, "Listening on: %s://%s\n", scheme, oneshotfmt.Address(host, port))
}

type clientSession struct {
	Request *events.HTTPRequest `json:",omitempty"`
	File    *events.File        `json:",omitempty"`
}

type report struct {
	Success  *clientSession
	Attempts []*clientSession
}
