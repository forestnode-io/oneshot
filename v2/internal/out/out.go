package out

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/raphaelreyna/oneshot/v2/internal/network"
	"github.com/raphaelreyna/oneshot/v2/internal/summary"
)

var nopTransferProgress TransferProgress = func(w io.Writer) *transferInfo {
	return nil
}

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

	cls                  []*clientSession
	currentClientSession *clientSession

	doneChan chan struct{}
}

var o = out{
	Stdout:   os.Stdout,
	doneChan: make(chan struct{}),
	cls:      make([]*clientSession, 0),
}

func (o *out) run() {
	switch o.Format {
	case "":
		o.runDefault()
	case "json":
		NewHTTPRequest = newHTTPRequest_WithBody
		o.runJSON()
	}

	o.doneChan <- struct{}{}
}

func (o *out) runJSON() {
	for event := range o.Events {
		switch event := event.(type) {
		case ClientDisconnected:
			o.cls = append(o.cls, o.currentClientSession)
			o.currentClientSession = nil
		case *HTTPRequest:
			o.currentClientSession = &clientSession{
				Request: event,
			}
		case *File:
			if o.currentClientSession != nil {
				o.currentClientSession.File = event
				if bf, ok := o.currentClientSession.File.Content.(func() []byte); ok {
					o.currentClientSession.File.Content = bf()
				}
			}
		case Success:
			if err := o.currentClientSession.Request.readBody(); err != nil {
				panic(err)
			}
			for _, s := range o.cls {
				if err := s.Request.readBody(); err != nil {
					panic(err)
				}
			}
			_ = json.NewEncoder(os.Stdout).Encode(report{
				Success:  o.currentClientSession,
				Attempts: o.cls,
			})
		default:
		}
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

			o.currentClientSession = &clientSession{
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
		default:
		}
	}
}

func (o *out) writeListeningOnQRCode(scheme, host, port string) {
	if o.Format == "json" || o.skipSummary {
		return
	}

	if host == "" {
		addrs, err := network.HostAddresses()
		if err != nil {
			addr := fmt.Sprintf("%s://localhost%s", scheme, port)
			fmt.Fprintf(o.Stdout, "%s:\n", addr)
			qrterminal.Generate(addr, qrterminal.L, o.Stdout)
			return
		}

		fmt.Fprintln(o.Stdout, "listening on: ")
		for _, addr := range addrs {
			addr = fmt.Sprintf("%s://%s", scheme, address(addr, port))
			fmt.Fprintf(o.Stdout, "%s:\n", addr)
			qrterminal.Generate(addr, qrterminal.L, o.Stdout)
		}
		return
	}

	addr := fmt.Sprintf("%s://%s", scheme, address(host, port))
	fmt.Fprintf(o.Stdout, "%s:\n", addr)
	qrterminal.Generate(addr, qrterminal.L, o.Stdout)
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

type clientSession struct {
	Request      *HTTPRequest  `json:",omitempty"`
	File         *File         `json:",omitempty"`
	TransferInfo *transferInfo `json:",omitempty"`
}

type transferInfo struct {
	WriteSize           int64         `json:",omitempty"`
	WriteStartTime      time.Time     `json:",omitempty"`
	WriteEndTime        time.Time     `json:",omitempty"`
	WriteDuration       time.Duration `json:",omitempty"`
	WriteBytesPerSecond int64         `json:",omitempty"`
}

type report struct {
	Success  *clientSession
	Attempts []*clientSession
}
