package output

import (
	"context"
	"log"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
)

func runQuiet(ctx context.Context, o *output) {
	for {
		select {
		case <-ctx.Done():
			if err := events.GetCancellationError(ctx); err != nil {
				log.Printf("connection cancellation error: %s", err.Error())
			}
			return
		case event := <-o.events:
			switch event := event.(type) {
			case *events.File:
				if bf, ok := event.Content.(func() []byte); ok && bf != nil {
					_ = bf()
				}
			case events.HTTPRequestBody:
				_, _ = event()
			}
		}
	}
}
