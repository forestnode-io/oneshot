package webrtc

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

const defaultDataChannelName = "oneshot"

type dataChannel struct {
	handler http.HandlerFunc
	errChan chan<- error
	*webrtc.DataChannel
}

func newDataChannel(name string, pc *peerConnection, handler http.HandlerFunc) (*dataChannel, <-chan error) {
	if name == "" {
		name = defaultDataChannelName
	}

	errs := make(chan error, 1)

	dc, err := pc.CreateDataChannel(name, nil)
	if err != nil {
		errs <- fmt.Errorf("unable to create data channel for webRTC peer connection: %w", err)
		return nil, errs
	}

	d := &dataChannel{
		DataChannel: dc,
		handler:     handler,
		errChan:     errs,
	}

	dc.OnOpen(d.onOpen)
	dc.OnClose(d.onClose)
	dc.OnMessage(d.onMessage)
	dc.OnError(d.onError)

	return d, errs
}

func (d *dataChannel) onOpen() {
	log.Println("data channel opened")
}

func (d *dataChannel) onClose() {
	log.Println("data channel closed")
	d.error(fmt.Errorf("data channel closed"))
}

func (d *dataChannel) onError(err error) {
	log.Println("data channel error:", err)
	d.error(err)
}

func (d *dataChannel) onMessage(msg webrtc.DataChannelMessage) {
	log.Println("OnMessage")

	buf := bufio.NewReader(bytes.NewBuffer(msg.Data))
	r, err := http.ReadRequest(buf)
	if err != nil {
		err = fmt.Errorf("unable to read request: %w", err)
		d.error(err)
	}

	w := httpResponseWriter{}
	d.handler(&w, r)

	if err := d.DataChannel.SendText(w.buf.String()); err != nil {
		err = fmt.Errorf("unable to send response: %w", err)
		d.error(err)
	}
}

func (d *dataChannel) error(err error) {
	go func() {
		d.errChan <- newDataChannelError(err)
	}()
}

type dataChannelError struct {
	error
}

func (e *dataChannelError) Error() string {
	return e.error.Error()
}

func (e *dataChannelError) Unwrap() error {
	return e.error
}

func newDataChannelError(err error) *dataChannelError {
	return &dataChannelError{error: err}
}
