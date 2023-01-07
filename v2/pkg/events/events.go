package events

import (
	"context"
)

// Event represents events in oneshot that should be communicated to the user.
type Event interface {
	_event
}

type _event interface {
	isEvent()
}

type SetEventChanFunc func(context.Context, chan Event)

func RegisterEventListener(ctx context.Context, f SetEventChanFunc) {
	b := bndl(ctx)
	f(ctx, b.eventsChan)
}

type ClientDisconnected struct {
	Err error
}

func (ClientDisconnected) isEvent() {}

func (c ClientDisconnected) Error() string {
	return c.Err.Error()
}

type HTTPRequestBody func() ([]byte, error)

func (HTTPRequestBody) isEvent() {}

func WithEvents(ctx context.Context) context.Context {
	b := bundle{
		eventsChan: make(chan Event, 1),
	}
	ctx = context.WithValue(ctx, bundleKey{}, &b)
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	return ctx
}

func eventChan(ctx context.Context) chan Event {
	return bndl(ctx).eventsChan
}

func Success(ctx context.Context) {
	b := bndl(ctx)
	b.cancel()
	close(b.eventsChan)
	b.eventsChan = nil
	b.cancel = nil
	b.success = true
}

func Succeeded(ctx context.Context) bool {
	return bndl(ctx).success
}

func Raise(ctx context.Context, e Event) {
	eventChan(ctx) <- e
}

type bundleKey struct{}
type bundle struct {
	eventsChan chan Event
	cancel     func()
	err        error
	success    bool
}

func bndl(ctx context.Context) *bundle {
	b, ok := ctx.Value(bundleKey{}).(*bundle)
	if !ok {
		panic("missing event bundle missing from context")
	}
	return b
}

func GetCancellationError(ctx context.Context) error {
	b := bndl(ctx)
	return b.err
}
