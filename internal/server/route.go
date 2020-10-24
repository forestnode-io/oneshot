package server

import (
	"net/http"
	"sync"
	"sync/atomic"
)

// FailableHandler is an http.HandlerFunc that returns an error.
// oneshot uses this error to determine when to exit.
type FailableHandler func(w http.ResponseWriter, r *http.Request) error

type Route struct {
	Pattern         string
	Methods         []string
	HandlerFunc     FailableHandler
	DoneHandlerFunc http.HandlerFunc
	MaxOK           int64
	MaxRequests     int64

	reqCount int64
	okCount  int64

	sync.Mutex
}

func (r *Route) RequestCount() int64 {
	return atomic.LoadInt64(&r.reqCount)
}

func (r *Route) OkCount() int64 {
	return atomic.LoadInt64(&r.okCount)
}
