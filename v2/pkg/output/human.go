package output

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
)

func runHuman(ctx context.Context, o *output) {
	for event := range o.events {
		switch event := event.(type) {
		case *events.ClientDisconnected:
			o.cls = append(o.cls, o.currentClientSession)
			o.currentClientSession = nil
		case *events.File:
			o.currentClientSession.File = event
			if bf, ok := o.currentClientSession.File.Content.(func() []byte); ok && bf != nil {
				if o.cmdName == "reverse-proxy" {
					os.Stdout.Write(bf())
					if o.stdoutTTY != nil {
						os.Stdout.Write([]byte("\n"))
					}
				} else {
					_ = bf()
				}
			}
		case *events.HTTPRequest:
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
	_human_handleContextDone(ctx, o)
}

func _human_handleContextDone(ctx context.Context, o *output) {
	if err := events.GetCancellationError(ctx); err != nil {
		log.Printf("connection cancellation error: %s", err.Error())
	}
}
