package webrtc

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/pion/datachannel"
)

type httpResponseWriter struct {
	header     http.Header
	sentHeader bool
	statusCode int

	channel        datachannel.ReadWriteCloser
	bufferedAmount func() int
	continueChan   chan struct{}
}

func newHTTPResponseWriter(dc *dataChannel) *httpResponseWriter {
	return &httpResponseWriter{
		channel:        dc.ReadWriteCloser,
		continueChan:   dc.continueChan,
		bufferedAmount: func() int { return int(dc.dc.BufferedAmount()) },
	}
}

func (w *httpResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *httpResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *httpResponseWriter) Write(b []byte) (int, error) {
	var err error
	if err = w.writeHeader(); err != nil {
		return 0, err
	}

	var (
		total int
		size  = dataChannelMTU
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
		if ba := w.bufferedAmount(); maxBufferedAmount < ba+size {
			<-w.continueChan
		}
	}

	return total, nil
}

func (w *httpResponseWriter) writeHeader() error {
	if w.sentHeader {
		return nil
	}
	w.sentHeader = true

	status := bytes.NewBuffer(nil)
	fmt.Fprintf(status, "HTTP/1.1 %d %s\n", w.statusCode, http.StatusText(w.statusCode))
	for k, v := range w.header {
		fmt.Fprintf(status, "%s: %s\n", k, v[0])
	}

	var (
		buf     = make([]byte, dataChannelMTU)
		payload = status.Bytes()
		mtu     = dataChannelMTU
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
