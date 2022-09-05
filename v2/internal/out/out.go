package out

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/raphaelreyna/oneshot/v2/internal/summary"
)

type Event interface {
	_event
}

type Out struct {
	Events <-chan Event
	Stdout io.Writer
	format string

	cls                  []*ClientSession
	currentClientSession *ClientSession
}

func (o *Out) Run() {
	o.cls = make([]*ClientSession, 0)
	switch o.format {
	case "":
		o.runDefault()
	case "json":
		NewHTTPRequest = newHTTPRequest_WithBody
	default:
	}
}

func (o *Out) runDefault() {
	for event := range o.Events {
		switch event := event.(type) {
		case ClientDisconnected:
			o.cls = append(o.cls, o.currentClientSession)
			o.currentClientSession = nil
			fmt.Println("\tClient disconnected")
		case Success:
			fmt.Println("Done")
		case *HTTPRequest:
			o.currentClientSession = &ClientSession{
				Request: event,
			}
			fmt.Printf("New connection from %v\n", event.RemoteAddr)
			fmt.Fprintf(o.Stdout, "\tRequest:\n\t\tURL: %v\n\t\tRequest URI: %v\n", event.URL, event.RequestURI)
		case HTTPRequestBody:
			body, err := event()
			if err != nil {
				panic(err)
			}
			o.currentClientSession.Request.Body = body
			fmt.Fprintln(os.Stdout, string(body))
		case TransferProgress:
			ti := event(o.Stdout)
			o.currentClientSession.TransferInfo = ti

			fmt.Fprintf(o.Stdout, "\tTransfer Info:\n")
			fmt.Fprintf(o.Stdout, "\t\tSize: %s\n", summary.PrettySize(ti.WriteSize))
			fmt.Fprintf(o.Stdout, "\t\tDuration: %v\n", ti.WriteDuration)
			fmt.Fprintf(o.Stdout, "\t\tRate: %s\n", summary.PrettyRate(ti.WriteBytesPerSecond))
		}
	}
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
