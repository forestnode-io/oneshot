package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/log"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	oneshotwebrtc "github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
)

type dataChannelEvent struct {
	err     error
	request *http.Request
}

type dataChannel struct {
	dc *webrtc.DataChannel
	datachannel.ReadWriteCloser
	continueChan chan struct{}

	eventsChan chan dataChannelEvent
	cancel     func()
}

func newDataChannel(ctx context.Context, pc *peerConnection) (*dataChannel, error) {
	log := log.Logger()
	dcChan := make(chan datachannel.ReadWriteCloser, 1)

	dc, err := pc.CreateDataChannel(oneshotwebrtc.DataChannelName, nil)
	if err != nil {
		err = fmt.Errorf("unable to create data channel for webRTC peer connection: %w", err)
		return nil, err
	}
	dc.SetBufferedAmountLowThreshold(oneshotwebrtc.BufferedAmountLowThreshold)

	d := &dataChannel{
		dc:           dc,
		continueChan: make(chan struct{}, 1),
		eventsChan:   make(chan dataChannelEvent, 1),
	}

	dc.OnClose(d.onClose)
	dc.OnError(d.onError)
	dc.OnOpen(func() {
		log.Debug().
			Msg("data channel opened")

		rawDC, err := dc.Detach()
		if err != nil {
			err = fmt.Errorf("unable to detach data channel for webRTC peer connection: %w", err)
			d.eventsChan <- dataChannelEvent{err: err}
			close(d.eventsChan)
			return
		}
		dcChan <- rawDC
		close(dcChan)
	})
	dc.OnBufferedAmountLow(func() {
		log.Debug().
			Msg("data channel buffered amount low")
		d.continueChan <- struct{}{}
	})

	// wait for the data channel to be established and detached (or an error)
	timedCtx, cancelTimedCtx := context.WithTimeout(ctx, 3*time.Second)
	defer cancelTimedCtx()
	select {
	case <-timedCtx.Done():
		return nil, timedCtx.Err()
	case e := <-d.eventsChan:
		if e.err != nil {
			close(d.eventsChan)
			return nil, fmt.Errorf("unable to establish data channel: %w", e.err)
		}
	case rawDC := <-dcChan:
		d.ReadWriteCloser = rawDC
	}

	preferredAddress, preferredPort := oneshotnet.PreferNonPrivateIP(pc.getPeerAddresses())
	remoteAddr := ""
	if preferredAddress != "" {
		remoteAddr = net.JoinHostPort(preferredAddress, preferredPort)
	}

	// start http request pump.
	// client can send fragmented http requests.
	// the client will send the head as a string and the body as binary.
	// an empty string signals the end of the request.
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	go func() {
		for {
			if ctx.Err() != nil {
				d.ReadWriteCloser.Close()
				close(d.eventsChan)
				return
			}

			var (
				buf               = make([]byte, oneshotwebrtc.DataChannelMTU)
				headerBuf         = bytes.NewBuffer(nil)
				doneReadingHeader = false
			)

			// read the HTTP request status line and header.
			for !doneReadingHeader {
				n, isString, err := d.ReadWriteCloser.ReadDataChannel(buf)
				if err != nil {
					d.ReadWriteCloser.Close()
					d.eventsChan <- dataChannelEvent{err: fmt.Errorf("unable to read data channel: %w", err)}
					return
				}
				if !isString {
					log.Error().
						Str("remote_addr", remoteAddr).
						Msg("received binary data during header parsing")
					d.ReadWriteCloser.Close()
					d.eventsChan <- dataChannelEvent{err: fmt.Errorf("received binary data during header parsing")}
					return
				}

				// if the client stopped sending string data, we have reached the end of the header.
				headerBreakIdx := bytes.Index(buf[:n], []byte("\n\n"))
				if headerBreakIdx != -1 {
					n = headerBreakIdx + 2
					_, _ = headerBuf.Write(buf[:n])
					doneReadingHeader = true
				} else {
					_, _ = headerBuf.Write(buf[:n])
				}
			}

			// create the HTTP request from the header.
			req, err := http.ReadRequest(bufio.NewReader(headerBuf))
			if err != nil {
				log.Error().Err(err).
					Msg("unable to read request")
				d.ReadWriteCloser.Close()
				d.eventsChan <- dataChannelEvent{err: err}
				return
			}
			req.RemoteAddr = remoteAddr

			// create the request body as a reader that reads from
			// the data channel until the client sends a string message
			b := newBody(d)
			req.Body = b
			d.eventsChan <- dataChannelEvent{request: req}

			// wait for the client to finish reading the body or the context to be canceled.
			select {
			case <-ctx.Done():
				return
			case <-b.doneChan:
				continue
			}
		}
	}()

	return d, nil
}

func (d *dataChannel) Close() error {
	return d.dc.Close()
}

func (d *dataChannel) onClose() {
	log := log.Logger()
	log.Debug().
		Msg("data channel closed")

	d.error(fmt.Errorf("data channel closed"))
	d.cancel()
	close(d.eventsChan)
	close(d.continueChan)
}

func (d *dataChannel) onError(err error) {
	log := log.Logger()
	log.Error().Err(err).
		Msg("data channel error")
	d.error(err)
}

func (d *dataChannel) error(err error) {
	go func() {
		d.eventsChan <- dataChannelEvent{err: err}
	}()
}

type body struct {
	buf      *bufio.Reader
	r        *oneshotwebrtc.DataChannelByteReader
	doneChan chan error
	done     bool
}

func newBody(dcRaw datachannel.ReadWriteCloser) *body {
	b := body{
		r:        &oneshotwebrtc.DataChannelByteReader{ReadWriteCloser: dcRaw},
		doneChan: make(chan error, 1),
	}
	b.buf = bufio.NewReaderSize(b.r, oneshotwebrtc.DataChannelMTU)
	return &b
}

func (b *body) Read(p []byte) (n int, err error) {
	if b.done {
		b.doneChan <- nil
		close(b.doneChan)
		b.doneChan = nil
		return 0, io.EOF
	}
	n, err = b.buf.Read(p)
	if err == io.EOF {
		b.done = true
		err = nil
	}
	return n, err
}

func (b *body) Close() error {
	if b.doneChan != nil {
		b.doneChan <- nil
		close(b.doneChan)
	}
	return nil
}
