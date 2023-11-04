package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pion/datachannel"
	"github.com/forestnode-io/oneshot/v2/pkg/net/webrtc"
)

type ResponseWriter struct {
	header     http.Header
	sentHeader bool
	statusCode int
	wroteCount int64

	channel        datachannel.ReadWriteCloser
	bufferedAmount func() int
	continueChan   chan struct{}

	triggersShutdown bool
}

func NewResponseWriter(dc *dataChannel) *ResponseWriter {
	return &ResponseWriter{
		channel:        dc.ReadWriteCloser,
		continueChan:   dc.continueChan,
		bufferedAmount: func() int { return int(dc.dc.BufferedAmount()) },
	}
}

func (w *ResponseWriter) TriggersShutdown() {
	w.triggersShutdown = true
}

func (w *ResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	var err error
	if err = w.writeHeader(); err != nil {
		return 0, err
	}

	var (
		total int
		size  = webrtc.DataChannelMTU
	)
	for i := 0; i < len(b); i += size {
		if maxSize := len(b) - i; maxSize < size {
			size = maxSize
		}

		n, err := w.channel.Write(b[i : i+size])
		total += n
		if err != nil {
			return total, err
		}

		// flow control
		// wait until the buffered amount (plus what we would send) is less than the maxBufferedAmount
		if ba := w.bufferedAmount(); webrtc.MaxBufferedAmount < ba+size {
			<-w.continueChan
		}
	}

	w.wroteCount += int64(total)

	return total, nil
}

func (w *ResponseWriter) Flush() error {
	err := w.writeHeader()
	if err != nil {
		return err
	}

	if w.wroteCount == 0 {
		// if we haven't written anything, send an empty binary message before the EOF
		_, err = w.channel.WriteDataChannel([]byte(""), false)
		if err != nil {
			return fmt.Errorf("unable to send empty binary message: %w", err)
		}
	}

	// send the EOF
	if _, err := w.channel.WriteDataChannel([]byte(""), true); err != nil {
		return fmt.Errorf("unable to send EOF: %w", err)
	}

	errs := make(chan error, 1)
	// wait for the eof ack
	go func() {
		eofBuf := make([]byte, 3)
		_, err = w.channel.Read(eofBuf)
		if err != nil {
			if err == io.EOF {
				err = nil
			} else {
				err = fmt.Errorf("unable to read EOF ACK: %w", err)
			}
		}
		errs <- err
	}()

	select {
	case err = <-errs:
		if err != nil {
			if strings.Contains(err.Error(), "Closed") ||
				errors.Is(err, io.EOF) {
				err = nil
			}
		}
		return err
	case <-time.After(333 * time.Millisecond):
		return nil
	}
}

func (w *ResponseWriter) writeHeader() error {
	if w.sentHeader {
		return nil
	}
	w.sentHeader = true

	status := bytes.NewBuffer(nil)
	fmt.Fprintf(status, "HTTP/1.1 %d %s\n", w.statusCode, http.StatusText(w.statusCode))
	for k, v := range w.header {
		fmt.Fprintf(status, "%s: %s\n", k, v[0])
	}
	fmt.Fprint(status, "\n")

	var (
		buf     = make([]byte, webrtc.DataChannelMTU)
		payload = status.Bytes()
		mtu     = webrtc.DataChannelMTU
		err     error
	)
	for 0 < len(payload) {
		if len(payload) < mtu {
			mtu = len(payload)
		}

		buf, payload = payload[:mtu], payload[mtu:]
		if _, err = w.channel.WriteDataChannel(buf[:mtu], true); err != nil {
			return err
		}
	}

	return nil
}
