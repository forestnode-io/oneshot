package events

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

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

// ReadBody reads in the http requests body by calling body() if its not nil.
// the body func just reads in a buffered copy of the body; it will have already
// been read from the client point of view.
func (hr *HTTPRequest) ReadBody() error {
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

func NewHTTPRequest(r *http.Request) *HTTPRequest {
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
func NewHTTPRequest_WithBody(r *http.Request) *HTTPRequest {
	ht := NewHTTPRequest(r)
	buf := bytes.NewBuffer(nil)
	r.Body = io.NopCloser(io.TeeReader(r.Body, buf))
	ht.body = func() ([]byte, error) {
		return io.ReadAll(buf)
	}
	return ht
}

func (*HTTPRequest) isEvent() {}

type URL struct {
	Scheme   string              `json:",omitempty"`
	User     string              `json:",omitempty"`
	Host     string              `json:",omitempty"`
	Path     string              `json:",omitempty"`
	Fragment string              `json:",omitempty"`
	Query    map[string][]string `json:",omitempty"`
}

func newURL(u *url.URL) URL {
	return URL{
		Scheme:   u.Scheme,
		User:     u.User.String(),
		Host:     u.Host,
		Path:     u.Path,
		Fragment: u.Fragment,
		Query:    u.Query(),
	}
}
