package webrtc

import (
	"fmt"
	"log"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
)

type dataChannel struct {
	errChan chan<- error
	dc      *webrtc.DataChannel
	datachannel.ReadWriteCloser
	continueChan chan struct{}
}

func newDataChannel(pc *peerConnection) (*dataChannel, chan error) {
	type dcOrErr struct {
		dc  *dataChannel
		err error
	}

	errs := make(chan error, 1)
	dcChan := make(chan datachannel.ReadWriteCloser, 1)

	dc, err := pc.CreateDataChannel(dataChannelName, nil)
	if err != nil {
		err = fmt.Errorf("unable to create data channel for webRTC peer connection: %w", err)
		errs <- err
		close(errs)
		return nil, errs
	}
	dc.SetBufferedAmountLowThreshold(bufferedAmountLowThreshold)

	d := &dataChannel{
		dc:           dc,
		errChan:      errs,
		continueChan: make(chan struct{}, 1),
	}

	dc.OnClose(d.onClose)
	dc.OnMessage(d.onMessage)
	dc.OnError(d.onError)
	dc.OnOpen(func() {
		log.Println("data channel opened")

		draw, err := dc.Detach()
		if err != nil {
			err = fmt.Errorf("unable to detach data channel for webRTC peer connection: %w", err)
			errs <- err
			return
		}
		dcChan <- draw
	})
	dc.OnBufferedAmountLow(func() {
		log.Println("data channel buffered amount low")
		d.continueChan <- struct{}{}
	})

	select {
	case err := <-errs:
		errs <- err
		close(errs)
		return nil, errs
	case draw := <-dcChan:
		d.ReadWriteCloser = draw
		return d, nil
	}
}

func (d *dataChannel) Close() error {
	return d.dc.Close()
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
