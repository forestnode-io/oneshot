package out

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/raphaelreyna/oneshot/v2/internal/network"
	"github.com/raphaelreyna/oneshot/v2/internal/summary"
)

type Event interface {
	_event
}

type out struct {
	Events     <-chan Event
	Stdout     io.Writer
	Format     string
	FormatOpts []string

	skipSummary     bool
	servingToStdout bool
	receivedBuf     *bytes.Buffer

	cls                  []*ClientSession
	currentClientSession *ClientSession
}

var o = out{
	Stdout: os.Stdout,
}

func (o *out) run() {
	o.cls = make([]*ClientSession, 0)
	switch o.Format {
	case "":
		o.runDefault()
	case "json":
		// TODO(raphaelreyna): currently the download process gets stuck on waiting for  events to be consumed.
		// create a json run mode that will consume and intergrate events into a json report at the end
		NewHTTPRequest = newHTTPRequest_WithBody
		o.runJSON()
	default:
	}
}

var nopTransferProgress TransferProgress = func(w io.Writer) *TransferInfo {
	return nil
}

func (o *out) runJSON() {
	for _ = range o.Events {
	}
}

func (o *out) runDefault() {
	for event := range o.Events {
		switch event := event.(type) {
		case ClientDisconnected:
			o.cls = append(o.cls, o.currentClientSession)
			o.currentClientSession = nil
			fmt.Println("\tClient disconnected")
		case Success:
			if o.servingToStdout {
				continue
			}

			fmt.Println("Done")
		case *HTTPRequest:
			if o.servingToStdout {
				continue
			}

			o.currentClientSession = &ClientSession{
				Request: event,
			}
			fmt.Fprintf(o.Stdout, "New connection from %v\n", event.RemoteAddr)
			fmt.Fprintf(o.Stdout, "\tRequest:\n")
			fmt.Fprintf(o.Stdout, "\t\tPath: %v\n", event.RequestURI)
			if len(event.URL.Query) != 0 {
				fmt.Fprintf(o.Stdout, "\t\tQuery:\n")
				for k, v := range event.URL.Query {
					fmt.Fprintf(o.Stdout, "\t\t\t- %s: ", k)
					if len(v) == 0 {
						fmt.Fprintln(o.Stdout, v)
					} else {
						fmt.Fprintln(o.Stdout, v[0])
					}
				}
			}
		case HTTPRequestBody:
			body, err := event()
			if err != nil {
				panic(err)
			}
			o.currentClientSession.Request.Body = body
			fmt.Fprintln(os.Stdout, string(body))
		case TransferProgress:
			if o.servingToStdout {
				continue
			}

			ti := event(o.Stdout)
			o.currentClientSession.TransferInfo = ti

			fmt.Fprintf(o.Stdout, "\tTransfer Info:\n")
			fmt.Fprintf(o.Stdout, "\t\tSize: %s\n", summary.PrettySize(ti.WriteSize))
			fmt.Fprintf(o.Stdout, "\t\tDuration: %v\n", ti.WriteDuration)
			fmt.Fprintf(o.Stdout, "\t\tRate: %s\n", summary.PrettyRate(ti.WriteBytesPerSecond))
		}
	}
}

func (o *out) writeListeningOn(scheme, host, port string) {
	if o.Format == "json" || o.skipSummary {
		return
	}

	if host == "" {
		addrs, err := network.HostAddresses()
		if err != nil {
			fmt.Fprintf(o.Stdout, "listening on: %s://localhost%s\n", scheme, port)
			return
		}

		fmt.Fprintln(o.Stdout, "listening on: ")
		for _, addr := range addrs {
			fmt.Fprintf(o.Stdout, "\t- %s://%s\n", scheme, address(addr, port))
		}
		return
	}

	fmt.Fprintf(o.Stdout, "listening on: %s://%s\n", scheme, address(host, port))
}

func address(host, port string) string {
	if port != "" {
		port = ":" + port
	}

	return host + port
}

type ClientSession struct {
	Request      *HTTPRequest  `json:",omitempty"`
	File         *File         `json:",omitempty"`
	TransferInfo *TransferInfo `json:",omitempty"`
}

type TransferInfo struct {
	WriteSize           int64         `json:",omitempty"`
	WriteStartTime      time.Time     `json:",omitempty"`
	WriteEndTime        time.Time     `json:",omitempty"`
	WriteDuration       time.Duration `json:",omitempty"`
	WriteBytesPerSecond int64         `json:",omitempty"`
}
