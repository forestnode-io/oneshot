package server

import "net/http"

type Middleware func(http.HandlerFunc) http.HandlerFunc

func (mw Middleware) Chain(m Middleware) Middleware {
	if mw == nil {
		return m
	}
	return func(hf http.HandlerFunc) http.HandlerFunc {
		hf = mw(hf)
		return m(hf)
	}
}

func demux(queueSize int64, next http.HandlerFunc) (http.HandlerFunc, func()) {
	type _wr struct {
		w    http.ResponseWriter
		r    *http.Request
		done func()
	}

	requestsQueue := make(chan _wr, queueSize)

	go func() {
		for wr := range requestsQueue {
			next(wr.w, wr.r)
			wr.done()
		}
	}()

	mw := func(w http.ResponseWriter, r *http.Request) {
		doneChan := make(chan struct{})
		wr := _wr{
			w: w,
			r: r,
			done: func() {
				close(doneChan)
			},
		}

		requestsQueue <- wr
		<-doneChan
	}

	return mw, func() {
		close(requestsQueue)
	}
}
