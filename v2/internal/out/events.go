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

type TransferProgress func(io.Writer) *transferInfo

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

// readBody reads in the http requests body by calling body() if its not nil.
// the body func just reads in a buffered copy of the body; it will have already
// been read from the client point of view.
func (hr *HTTPRequest) readBody() error {
	bf := hr.body
	if bf == nil || hr.Body != nil {
		return nil
	}

	body, err := bf()
	if err != nil {
		return err
	}

	hr.Body = body

	return nil
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

// newHTTPRequest_WithBody replaces the requests body with a tee reader that copies the data into a byte buffer.
// This allows for the body to be written out later in a report should we need to.
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
	Name    string `json:",omitempty"`
	Path    string `json:",omitempty"`
	MIME    string `json:",omitempty"`
	Size    int64  `json:",omitempty"`
	Content any    `json:",omitempty"`
}

func (*File) isEvent() {}
