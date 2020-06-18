package server

import (
	"net/http"
	"sync"
	"sync/atomic"
)

type Route struct {
	Pattern         string
	Methods         []string
	HandlerFunc     func(w http.ResponseWriter, r *http.Request) error
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
