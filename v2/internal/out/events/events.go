package events

import "io"

type Event interface {
	_event
}

type _event interface {
	isEvent()
}

type ClientDisconnected struct {
	Err error
}

func (ClientDisconnected) isEvent() {}

type Success struct{}

func (Success) isEvent() {}

type HTTPRequestBody func() ([]byte, error)

func (HTTPRequestBody) isEvent() {}

type TransferProgress func(io.Writer)

func (TransferProgress) isEvent() {}