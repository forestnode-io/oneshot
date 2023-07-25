package output

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/forestnode-io/oneshot/v2/pkg/events"
	"github.com/rs/zerolog"
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
	log := zerolog.Ctx(ctx)
	if err := events.GetCancellationError(ctx); err != nil {
		log.Error().Err(err).
			Msg("connection cancelled event")
	}
}

type PrintableError struct {
	Err error
}

func (h *PrintableError) Error() string {
	return fmt.Sprintf("%v", h.Err)
}

func (h *PrintableError) Unwrap() error {
	return h.Err
}

func WrapPrintable(err error) error {
	if err == nil {
		return nil
	}
	return &PrintableError{Err: err}
}

func IsPrintable(err error) bool {
	var e *PrintableError
	return errors.As(err, &e)
}
