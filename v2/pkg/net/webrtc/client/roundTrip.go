package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/pion/datachannel"
	oneshotwebrtc "github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
	"golang.org/x/net/http/httpguts"
)

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	bundle := <-t.dcChan
	if bundle.err != nil {
		return nil, fmt.Errorf("unable to get data channel: %w", bundle.err)
	}
	raw := bundle.raw
	closeBody := func() {
		if req.Body != nil {
			if err := req.Body.Close(); err != nil {
				log.Printf("unable to close request body: %v", err)
			}
		}
	}

	if req.URL == nil {
		closeBody()
		return nil, fmt.Errorf("invalid request: missing URL")
	}
	if req.Header == nil {
		closeBody()
		return nil, fmt.Errorf("invalid request: missing header")
	}

	for k, vv := range req.Header {
		if !httpguts.ValidHeaderFieldName(k) {
			closeBody()
			return nil, fmt.Errorf("net/http: invalid header field name %q", k)
		}
		for _, v := range vv {
			if !httpguts.ValidHeaderFieldValue(v) {
				closeBody()
				// Don't include the value in the error, because it may be sensitive.
				return nil, fmt.Errorf("net/http: invalid header field value for %q", k)
			}
		}
	}

	headerBuf := bytes.NewBuffer(nil)
	path := req.URL.RequestURI()
	fmt.Fprintf(headerBuf, "%s %s %s\n", req.Method, path, req.Proto)
	for k, v := range req.Header {
		fmt.Fprintf(headerBuf, "%s: %s\n", k, v)
	}
	fmt.Fprintf(headerBuf, "\n")

	if _, err := raw.WriteDataChannel(headerBuf.Bytes(), true); err != nil {
		return nil, fmt.Errorf("unable to send request header: %w", err)
	}

	var buf = make([]byte, oneshotwebrtc.DataChannelMTU)
	if req.Body != nil {
		var w = &flowControlledWriter{
			w:                 raw,
			bufferedAmount:    func() int { return int(bundle.dc.BufferedAmount()) },
			maxBufferedAmount: oneshotwebrtc.MaxBufferedAmount,
			continueChan:      t.continueChan,
		}

		if _, err := io.CopyBuffer(w, req.Body, buf); err != nil {
			return nil, fmt.Errorf("unable to send request body: %w", err)
		}
	}

	if _, err := raw.WriteDataChannel([]byte(""), true); err != nil {
		return nil, fmt.Errorf("unable to send request body EOF: %w", err)
	}

	r := dcReader{raw: raw}
	resp, err := http.ReadResponse(bufio.NewReaderSize(&r, oneshotwebrtc.DataChannelMTU), req)
	if err != nil {
		return nil, fmt.Errorf("unable to read response: %w", err)
	}
	return resp, nil
}

type dcReader struct {
	raw     datachannel.ReadWriteCloser
	hitBody bool
}

func (r *dcReader) Read(p []byte) (int, error) {
	n, isString, err := r.raw.ReadDataChannel(p)
	if isString {
		if r.hitBody {
			return n, io.EOF
		}
	} else {
		r.hitBody = true
	}

	return n, err
}
