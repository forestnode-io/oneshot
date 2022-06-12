package summary

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type File struct {
	Name string `json:",omitempty"`
	MIME string `json:",omitempty"`
	Size int64  `json:",omitempty"`
}

type Summary struct {
	StartTime         time.Time     `json:",omitempty"`
	EndTime           time.Time     `json:",omitempty"`
	Duration          time.Duration `json:",omitempty"`
	FailedRequests    []*Request    `json:",omitempty"`
	SuccessfulRequest *Request      `json:",omitempty"`
	SuccessfulFile    *File         `json:",omitempty"`

	sync.Mutex
}

func NewSummary(start time.Time) *Summary {
	return &Summary{
		StartTime:      start,
		FailedRequests: make([]*Request, 0),
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
	if file := r.File; file != nil {
		s.SuccessfulFile = file
	}
}

func (s *Summary) WriteJSON(w io.Writer, pretty bool) {
	je := json.NewEncoder(w)
	if pretty {
		je.SetIndent("", "\t")
	}
	je.Encode(s)
}

func (s *Summary) WriteHuman(w io.Writer) {
	if len(s.FailedRequests) == 0 && s.SuccessfulRequest == nil {
		return
	}

	s.Lock()
	defer s.Unlock()

	fmt.Fprintln(w, "transfer complete:")
	if file := s.SuccessfulFile; file != nil {
		fmt.Fprintf(w, "\tfile name: %s\n", file.Name)
		fmt.Fprintf(w, "\tMIME type: %s\n", file.MIME)
		fmt.Fprintf(w, "\tfile size: %s\n", PrettySize(file.Size))
	}

	if succReq := s.SuccessfulRequest; succReq != nil {
		fmt.Fprintf(w, "\tstart time: %v\n", succReq.StartTime)
		fmt.Fprintf(w, "\tduration: %+v\n", succReq.Duration)
		fmt.Fprintf(w, "\trate: %+v\n", nil)
		fmt.Fprintf(w, "\tclient address: %+v\n", succReq.RemoteAddr)
	}
}

type Request struct {
	Method     string              `json:",omitempty"`
	URL        URL                 `json:",omitempty"`
	Protocol   string              `json:",omitempty"`
	Header     map[string][]string `json:",omitempty"`
	Host       string              `json:",omitempty"`
	Trailer    map[string][]string `json:",omitempty"`
	RemoteAddr string              `json:",omitempty"`
	RequestURI string              `json:",omitempty"`
	StartTime  time.Time           `json:",omitempty"`
	EndTime    time.Time           `json:",omitempty"`
	Duration   time.Duration       `json:",omitempty"`

	Body any `json:",omitempty"`

	WriteSize           int64         `json:",omitempty"`
	WriteStartTime      time.Time     `json:",omitempty"`
	WriteEndTime        time.Time     `json:",omitempty"`
	WriteDuration       time.Duration `json:",omitempty"`
	WriteBytesPerSecond int64         `json:",omitempty"`
	File                *File         `json:",omitempty"`
}

func (r *Request) SetTimes(start, end time.Time) {
	r.StartTime = start
	r.EndTime = end
	r.Duration = end.Sub(start)
}

func (r *Request) SetWriteStats(w http.ResponseWriter) {
	rw, ok := w.(*responseWriter)
	if !ok {
		return
	}
	r.WriteSize = int64(rw.size)
	r.WriteStartTime = rw.start
	r.WriteEndTime = time.Now()
	r.WriteDuration = r.WriteEndTime.Sub(rw.start)
	r.WriteBytesPerSecond = r.WriteSize / int64(r.Duration*time.Second)
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

func (r *Request) contentType() string {
	if hdrs := r.Header["Content-Type"]; len(hdrs) != 0 {
		return hdrs[0]
	}

	return ""
}

func (r *Request) SetBody(body io.Reader) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return
	}

	var ct = r.contentType()
	switch {
	case ct == "application/json":
		r.Body = json.RawMessage(bodyBytes)
	case strings.Contains(ct, "text/"):
		r.Body = string(bodyBytes)
	default:
		r.Body = bodyBytes
	}
}

const (
	kb = 1000
	mb = kb * 1000
	gb = mb * 1000
)

func PrettySize(n int64) string {
	var (
		str  string
		size = float64(n)
	)

	// Create the size string using appropriate units: B, KB, MB, and GB
	switch {
	case size < kb:
		str = fmt.Sprintf("%d B", n)
	case size < mb:
		size = size / kb
		str = fmt.Sprintf("%.3f KB", size)
	case size < gb:
		size = size / mb
		str = fmt.Sprintf("%.3f MB", size)
	default:
		size = size / gb
		str = fmt.Sprintf("%.3f GB", size)
	}

	return str
}

func PrettyRate(n int64) string {
	var (
		str  string
		rate = float64(n)
	)

	// Create the size string using appropriate units: B, KB, MB, and GB
	switch {
	case rate < kb:
		str = fmt.Sprintf("%.3f B/s", rate)
	case rate < mb:
		rate = rate / kb
		str = fmt.Sprintf("%.3f KB/s", rate)
	case rate < gb:
		rate = rate / mb
		str = fmt.Sprintf("%.3f MB/s", rate)
	default:
		rate = rate / gb
		str = fmt.Sprintf("%.3f GB/s", rate)
	}

	return str
}

type responseWriter struct {
	http.ResponseWriter
	start time.Time
	size  int
}

func NewResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if rw.start.IsZero() {
		rw.start = time.Now()
	}
	n, err := rw.ResponseWriter.Write(p)
	rw.size += n

	return n, err
}
