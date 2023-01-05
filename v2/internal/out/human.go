package out

import (
	"fmt"
	"os"

	"github.com/raphaelreyna/oneshot/v2/internal/events"
)

func runHuman(o *output) {
	for event := range o.events {
		switch event := event.(type) {
		case *events.ClientDisconnected:
			if !o.servingToStdout {
				fmt.Fprintf(o.Stdout, "...disconnected\n")
			}

			o.cls = append(o.cls, o.currentClientSession)
			o.currentClientSession = nil
		case *events.File:
			o.currentClientSession.File = event
			if bf, ok := o.currentClientSession.File.Content.(func() []byte); ok && bf != nil {
				_ = bf()
			}
		case *events.HTTPRequest:
			if o.servingToStdout {
				continue
			}

			o.currentClientSession = &clientSession{
				Request: event,
			}
		case events.HTTPRequestBody:
			body, err := event()
			if err != nil {
				panic(err)
			}
			o.currentClientSession.Request.Body = body
			fmt.Fprintln(os.Stdout, string(body))
		default:
		}
	}
}
