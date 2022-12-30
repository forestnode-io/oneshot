package out

import (
	"bytes"
	"io"
	"net/http"
)

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

type TransferProgress func(io.Writer) *TransferInfo

func (TransferProgress) isEvent() {}

type HTTPRequest struct {
	Method     string              `json:",omitempty"`
	URL        URL                 `json:",omitempty"`
	Protocol   string              `json:",omitempty"`
	Header     map[string][]string `json:",omitempty"`
	Host       string              `json:",omitempty"`
	Trailer    map[string][]string `json:",omitempty"`
	RemoteAddr string              `json:",omitempty"`
	RequestURI string              `json:",omitempty"`

	Body any `json:",omitempty"`

	body func() ([]byte, error) `json:"-"`
}

var NewHTTPRequest = newHTTPRequest

func newHTTPRequest(r *http.Request) *HTTPRequest {
	return &HTTPRequest{
		Method:     r.Method,
		URL:        newURL(r.URL),
		Protocol:   r.Proto,
		Header:     r.Header,
		Host:       r.Host,
		Trailer:    r.Trailer,
		RemoteAddr: r.RemoteAddr,
		RequestURI: r.RequestURI,
	}
}

func newHTTPRequest_WithBody(r *http.Request) *HTTPRequest {
	ht := newHTTPRequest(r)
	buf := bytes.NewBuffer(nil)
	r.Body = io.NopCloser(io.TeeReader(r.Body, buf))
	ht.body = func() ([]byte, error) {
		return io.ReadAll(buf)
	}
	return ht
}

func (*HTTPRequest) isEvent() {}

type File struct {
	Name    string `json:"Name,omitempty"`
	Path    string `json:",omitempty"`
	MIME    string `json:",omitempty"`
	Size    int64  `json:",omitempty"`
	Content any    `json:",omitempty"`
}

func (*File) isEvent() {}
