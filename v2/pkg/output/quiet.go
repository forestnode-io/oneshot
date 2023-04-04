package output

import (
	"context"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/rs/zerolog"
)

func runQuiet(ctx context.Context, o *output) {
	log := zerolog.Ctx(ctx)

	for {
		select {
		case <-ctx.Done():
			if err := events.GetCancellationError(ctx); err != nil {
				log.Error().Err(err).
					Msg("connection cancelled event")
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
