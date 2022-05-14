package server

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"sync"

	"github.com/gorilla/mux"
)

type connKey struct{}

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request) (interface{}, error)
	ServeExpiredHTTP(http.ResponseWriter, *http.Request)
}

type _wr struct {
	w    http.ResponseWriter
	r    *http.Request
	done func()
}

type Server struct {
	http.Server

	requestsQueue chan _wr

	results          []interface{}
	successfulResult interface{}
	succResMu        sync.RWMutex
	stopWorkers      func()

	handler Handler
}

func NewServer(handler Handler) *Server {
	s := Server{
		requestsQueue: make(chan _wr),
		results:       make([]interface{}, 0),
		handler:       handler,
	}

	return &s
}

func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	r := mux.NewRouter()
	r.HandleFunc("/", s.rootHandler)

	s.Handler = r
	s.BaseContext = func(l net.Listener) context.Context {
		return ctx
	}
	s.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, connKey{}, c)
	}

	ctx, s.stopWorkers = context.WithCancel(ctx)

	go s.preTransferWorker(ctx)

	var errs = make(chan error)
	go func() {
		errs <- s.Server.Serve(l)
	}()

	// TODO(raphaelreyna) wait for transfer to succeed and drain client queue

	return <-errs
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.requestsQueue)
	s.stopWorkers()

	return s.Server.Shutdown(ctx)
}

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	// go dark if the transfer succeeded
	if s.transferSucceeded() {
		var (
			ctx  = r.Context()
			conn = ctx.Value(connKey{}).(net.Conn)
		)

		conn.Close()
		return
	}

	var (
		done = make(chan struct{})

		wr = _wr{
			w: w,
			r: r,
			done: func() {
				done <- struct{}{}
				close(done)
			},
		}
	)

	s.requestsQueue <- wr

	<-done
}

func (s *Server) preTransferWorker(ctx context.Context) {
	for {
		if s.transferSucceeded() {
			for i := 0; i < runtime.NumCPU(); i++ {
				go s.postTransferWorker(ctx)
			}
			return
		}

		select {
		case <-ctx.Done():
			return
		case wr, ok := <-s.requestsQueue:
			if !ok {
				return
			}

			iface, err := s.handler.ServeHTTP(wr.w, wr.r)
			if err != nil {
				s.results = append(s.results, iface)
			} else {
				s.succResMu.Lock()
				s.successfulResult = iface
				s.succResMu.Unlock()
			}

			wr.done()
		}
	}
}

func (s *Server) postTransferWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case wr, ok := <-s.requestsQueue:
			if !ok {
				return
			}

			s.handler.ServeExpiredHTTP(wr.w, wr.r)

			wr.done()
		}
	}
}

func (s *Server) transferSucceeded() bool {
	mu := &s.succResMu
	mu.RLock()
	defer mu.RUnlock()
	return s.successfulResult != nil
}
