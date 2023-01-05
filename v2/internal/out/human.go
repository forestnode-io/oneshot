package out

import (
	"fmt"
	"os"

	"github.com/raphaelreyna/oneshot/v2/internal/out/events"
)

func (o *out) runHuman() {
	var initialTransferProgress = true

	for event := range o.Events {
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
		case events.TransferProgress:
			if o.servingToStdout {
				continue
			}

			if initialTransferProgress {
				o.Stdout.SaveCursorPosition()
			}
			event(o.Stdout)
		default:
		}
	}
}
