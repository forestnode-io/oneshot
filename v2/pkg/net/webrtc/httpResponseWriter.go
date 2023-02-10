package webrtc

import (
	"bytes"
	"fmt"
	"net/http"
)

type httpResponseWriter struct {
	header     http.Header
	sentHeader bool

	buf *bytes.Buffer
}

func (w *httpResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *httpResponseWriter) WriteHeader(statusCode int) {
	if w.sentHeader {
		return
	}
	w.sentHeader = true

	status := fmt.Sprintf("HTTP/1.1 %d %s\n", statusCode, http.StatusText(statusCode))
	for k, v := range w.header {
		status += fmt.Sprintf("%s: %s\n", k, v[0])
	}
	status += "\n"

	if w.buf == nil {
		w.buf = new(bytes.Buffer)
	}
	w.buf.WriteString(status)
}

func (w *httpResponseWriter) Write(b []byte) (int, error) {
	if !w.sentHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.buf == nil {
		w.buf = new(bytes.Buffer)
	}
	return w.buf.Write(b)
}
