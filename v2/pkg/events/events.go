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
	ctx, cancel := context.WithCancel(ctx)
	b := bundle{
		eventsChan: make(chan Event, 1),
		cancel:     cancel,
		exitCode:   -1,
	}

	go func() {
		<-ctx.Done()
		close(b.eventsChan)
	}()
	ctx = context.WithValue(ctx, bundleKey{}, &b)

	return ctx
}

func eventChan(ctx context.Context) chan Event {
	return bndl(ctx).eventsChan
}

func Success(ctx context.Context) {
	b := bndl(ctx)
	b.success = true
}

func Succeeded(ctx context.Context) bool {
	return bndl(ctx).success
}

func Raise(ctx context.Context, e Event) {
	eventChan(ctx) <- e
}

func Stop(ctx context.Context) {
	b := bndl(ctx)
	b.cancel()
}

type bundleKey struct{}
type bundle struct {
	eventsChan chan Event
	err        error
	success    bool
	cancel     func()
	exitCode   int
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

func SetExitCode(ctx context.Context, code int) {
	b := bndl(ctx)
	b.exitCode = code
}

func GetExitCode(ctx context.Context) int {
	b := bndl(ctx)
	return b.exitCode
}
