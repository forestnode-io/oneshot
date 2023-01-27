package output

import (
	"io"
	"net/http"
)

type ResponseWriter struct {
	W         http.ResponseWriter
	BufferedW io.Writer

	wroteHeader bool
}

func (w *ResponseWriter) Header() http.Header {
	return w.W.Header()
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.W.WriteHeader(http.StatusOK)
		w.wroteHeader = true
	}
	return w.BufferedW.Write(p)
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.W.WriteHeader(statusCode)
	w.wroteHeader = true
}
