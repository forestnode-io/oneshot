package summary

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Summary struct {
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	FailedRequests    []*Request
	SuccessfulRequest *Request

	sync.Mutex
}

func NewSummary(start time.Time) *Summary {
	return &Summary{
		StartTime: start,
	}
}

func (s *Summary) End() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
}

func (s *Summary) Succesful() bool {
	s.Lock()
	defer s.Unlock()

	return s.SuccessfulRequest != nil
}

func (s *Summary) AddFailure(r *Request) {
	s.Lock()
	defer s.Unlock()

	if s.FailedRequests == nil {
		s.FailedRequests = make([]*Request, 0)
	}

	s.FailedRequests = append(s.FailedRequests, r)
}

func (s *Summary) SucceededWith(r *Request) {
	s.Lock()
	defer s.Unlock()

	s.SuccessfulRequest = r
}

type Request struct {
	Method                 string
	URL                    URL
	Protocol               string
	Header                 map[string][]string
	Host                   string
	Trailer                map[string][]string
	RemoteAddr             string
	RequestURI             string
	StartTime              time.Time
	EndTime                time.Time
	Duration               time.Duration
	TransferSize           int64
	TransferBytesPerSecond int64
}

func (r *Request) SetTimes(start, end time.Time) {
	r.StartTime = start
	r.EndTime = end
	r.Duration = end.Sub(start)
}

func NewRequest(r *http.Request) *Request {
	return &Request{
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

type URL struct {
	Scheme   string
	User     string
	Host     string
	Path     string
	Fragment string
	Query    map[string][]string
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
