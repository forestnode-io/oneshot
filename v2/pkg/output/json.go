package output

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
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
	includeFileContent = includeFileContent || o.ttyForContentOnly
	if _, exclude := o.FormatOpts["exclude-file-contents"]; exclude {
		includeFileContent = false
	}

	switch event := e.(type) {
	case *events.ClientDisconnected:
		o.cls = append(o.cls, o.currentClientSession)
		o.currentClientSession = nil
	case *events.HTTPRequest:
		o.currentClientSession = &clientSession{
			Request: event,
		}
	case *events.File:
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
	}
}

func _json_handleContextDone(ctx context.Context, o *output) {
	if err := events.GetCancellationError(ctx); err != nil {
		log.Printf("connection cancellation error: %s", err.Error())
	}

	// if serving to stdout
	if o.ttyForContentOnly {
		// then read in the body since it wasnt written to disk
		if err := o.currentClientSession.Request.ReadBody(); err != nil {
			log.Printf("error reading request body buffer: %s", err.Error())
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
		if o.ttyForContentOnly {
			// then read in the body since it wasnt written to disk
			if err := s.Request.ReadBody(); err != nil {
				log.Printf("error reading request body buffer: %s", err.Error())
			}
		} else {
			// otherwise, theres no point in showing the content again in stdout
			s.Request.Body = nil
		}
		s.File.ComputeTransferFields()
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
		log.Printf("error encoding json to stdout: %s", err.Error())
	}
}
