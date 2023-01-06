package out

import (
	"bytes"
	"context"
	"fmt"

	"github.com/mdp/qrterminal/v3"
	"github.com/muesli/termenv"
	"github.com/raphaelreyna/oneshot/v2/internal/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/internal/net"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/internal/out/fmt"
)

type key struct{}

func getOut(ctx context.Context) *output {
	o, _ := ctx.Value(key{}).(*output)
	if o == nil {
		panic("no output set")
	}
	return o
}

type output struct {
	events     chan events.Event
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

func (o *output) run(ctx context.Context) {
	if !o.servingToStdout {
		o.Stdout.HideCursor()
		defer o.Stdout.ShowCursor()
	}

	switch o.Format {
	case "":
		runHuman(o)
	case "json":
		NewHTTPRequest = events.NewHTTPRequest_WithBody
		runJSON(ctx, o)
	}

	o.doneChan <- struct{}{}
}

func (o *output) writeListeningOnQRCode(scheme, host, port string) {
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
		addrs, err := oneshotnet.HostAddresses()
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

type clientSession struct {
	Request *events.HTTPRequest `json:",omitempty"`
	File    *events.File        `json:",omitempty"`
}

type report struct {
	Success  *clientSession
	Attempts []*clientSession
}
