package client

import "io"

type flowControlledWriter struct {
	w                 io.Writer
	bufferedAmount    func() int
	maxBufferedAmount int
	continueChan      chan struct{}
}

func (w *flowControlledWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)

	// if we are over the max buffered amount, wait for the continue channel
	if ba := w.bufferedAmount(); w.maxBufferedAmount < ba+n {
		<-w.continueChan
	}

	return n, err
}
