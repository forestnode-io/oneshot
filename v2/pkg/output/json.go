package output

import (
	"context"
	"encoding/json"
	"os"

	"github.com/oneshot-uno/oneshot/v2/pkg/events"
	"github.com/rs/zerolog"
)

func runJSON(ctx context.Context, o *output) {
	for event := range o.events {
		_json_handleEvent(o, event)
	}
	_json_handleContextDone(ctx, o)
}

func _json_handleEvent(o *output, e events.Event) {
	_, includeFileContent := o.FormatOpts["include-file-contents"]
	// if the user want to receive to stdout then the received content wont be saved to disk
	// so we include it in the json report which is going to stdout
	if _, exclude := o.FormatOpts["exclude-file-contents"]; exclude {
		includeFileContent = false
	}

	switch event := e.(type) {
	case events.ClientDisconnected:
		if err := event.Err; err != nil {
			o.currentClientSession.Error = err.Error()
		}
		o.cls = append(o.cls, o.currentClientSession)
		o.currentClientSession = nil
	case *events.HTTPRequest:
		o.currentClientSession = &clientSession{
			Request: event,
		}
	case *events.File:
		if o.cmdName == "reverse-proxy" {
			if o.currentClientSession != nil {
				if o.currentClientSession.Response == nil {
					o.currentClientSession.Response = &events.HTTPResponse{}
				}

				if bf, ok := event.Content.(func() []byte); ok {
					o.currentClientSession.Response.Body = bf()
				}
			}

			return
		}

		if o.currentClientSession != nil {
			o.currentClientSession.File = event
			if bf, ok := o.currentClientSession.File.Content.(func() []byte); ok {
				if includeFileContent {
					o.currentClientSession.File.Content = bf()
					if o.currentClientSession.File.TransferSize == 0 {
						o.currentClientSession.File.TransferSize = int64(len(o.currentClientSession.File.Content.([]byte)))
					}
				} else {
					// otherwise, dump the contents
					_ = bf()
					o.currentClientSession.File.Content = nil
				}
			}
		}
	case *events.HTTPResponse:
		if o.currentClientSession != nil {
			if o.currentClientSession.Response != nil {
				if o.currentClientSession.Response.Body != nil {
					if body, ok := o.currentClientSession.Response.Body.([]byte); ok {
						event.Body = body
					}
				}
			}
			o.currentClientSession.Response = event
		}
	}
}

func _json_handleContextDone(ctx context.Context, o *output) {
	log := zerolog.Ctx(ctx)

	if err := events.GetCancellationError(ctx); err != nil {
		log.Error().
			Msg("connection cancelled event")
	}

	// if serving to stdout
	if o.includeBody {
		if o.currentClientSession != nil {
			if o.currentClientSession.Request != nil {
				// then read in the body since it wasnt written to disk
				if err := o.currentClientSession.Request.ReadBody(); err != nil {
					log.Error().Err(err).
						Msg("error reading request body buffer")
				}
			}
		}
	} else if o.currentClientSession != nil {
		if o.currentClientSession.Request != nil {
			// otherwise, theres no point in showing the content again in stdout
			o.currentClientSession.Request.Body = nil
		}
	}
	if o.currentClientSession != nil {
		if o.currentClientSession.File != nil {
			o.currentClientSession.File.ComputeTransferFields()
		}
	}

	for _, s := range o.cls {
		// if serving to stdout
		if o.includeBody {
			if s.Request != nil {
				// then read in the body since it wasnt written to disk
				if err := s.Request.ReadBody(); err != nil {
					log.Error().Err(err).
						Msg("error reading request body buffer")
				}
			}
		} else {
			// otherwise, theres no point in showing the content again in stdout
			s.Request.Body = nil
		}
		s.File.ComputeTransferFields()
	}

	if o.currentClientSession == nil && len(o.cls) == 0 {
		return
	}

	enc := json.NewEncoder(os.Stdout)
	if _, ok := o.FormatOpts["compact"]; !ok {
		enc.SetIndent("", "  ")
	}
	err := enc.Encode(Report{
		Success:  o.currentClientSession,
		Attempts: o.cls,
	})
	if err != nil {
		log.Error().Err(err).
			Msg("error encoding json to stdout")
	}
}
