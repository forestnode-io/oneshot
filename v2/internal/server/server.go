package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/raphaelreyna/oneshot/v2/internal/api"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
)

var ErrTimeout = errors.New("timeout")

type connKey struct{}

type _wr struct {
	w         http.ResponseWriter
	r         *http.Request
	done      func()
	startTime time.Time
}

type Server struct {
	http.Server

	requestsQueue chan _wr
	requestsWG    sync.WaitGroup

	success     bool
	succChan    chan struct{}
	timeoutChan chan struct{}
	stopWorkers func()

	serveHTTP        api.HTTPHandler
	serveExpiredHTTP api.HTTPHandler

	Events chan<- out.Event

	bufferRequestBody bool
	timeout           time.Duration
}

func NewServer(serve, serveExpired api.HTTPHandler) *Server {
	s := Server{
		requestsQueue:    make(chan _wr),
		serveHTTP:        serve,
		serveExpiredHTTP: serveExpired,
		succChan:         make(chan struct{}),
	}

	return &s
}

func (s *Server) AddMiddleware(m Middleware) {
	if m == nil {
		return
	}

	s.serveHTTP = m(s.serveHTTP)
	s.serveExpiredHTTP = m(s.serveExpiredHTTP)
}

func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(s.rootHandler)

	s.Handler = r
	s.BaseContext = func(l net.Listener) context.Context {
		return ctx
	}
	s.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, connKey{}, c)
	}

	ctx, s.stopWorkers = context.WithCancel(ctx)

	go s.preTransferWorker(ctx)

	if 0 < s.timeout {
		s.timeoutChan = make(chan struct{}, 1)
		l = withTimeout(s.timeout, s.timeoutChan, l)
	}
	var errs = make(chan error)
	go func() {
		errs <- s.Server.Serve(l)
	}()

	go func() {
		<-s.succChan
		s.requestsWG.Wait()
		errs <- nil
	}()

	go func() {
		<-s.timeoutChan
		fmt.Printf("timeout after %v; exiting ...\n", s.timeout)
		errs <- nil
	}()

	err := <-errs
	if err != nil {
		return err
	}

	s.Shutdown(ctx)
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.requestsQueue)
	s.stopWorkers()

	return s.Server.Close()
}

func (s *Server) BufferRequests() {
	s.bufferRequestBody = true
}

func (s *Server) SetTimeoutDuration(d time.Duration) {
	s.timeout = d
}

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	// go dark if the transfer succeeded
	if s.success {
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
			w:         w,
			r:         r,
			startTime: time.Now(),
			done: func() {
				done <- struct{}{}
				close(done)
			},
		}
	)

	s.requestsWG.Add(1)
	s.requestsQueue <- wr

	<-done
	s.requestsWG.Done()
}

func (s *Server) preTransferWorker(ctx context.Context) {
	for {
		// if there was a succesfull transfer,
		// startup as many postTransferWorkers as posssible
		// and end this worker to cleanly end the waiting connections asap.
		if s.success {
			s.succChan <- struct{}{}
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

			var (
				apiCtx = apiCtx{
					events: s.Events,
				}
			)

			s.serveHTTP(&apiCtx, wr.w, wr.r) // run the command server

			if apiCtx.success {
				s.success = true
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

			var apiCtx apiCtx
			s.serveExpiredHTTP(&apiCtx, wr.w, wr.r)

			wr.done()
		}
	}
}

type apiCtx struct {
	events  chan<- out.Event
	success bool
}

func (a *apiCtx) Success() {
	a.success = true
	a.events <- out.Success{}
}

func (a *apiCtx) Raise(e out.Event) {
	a.events <- e
}

type serverKey struct{}

func WithServer(ctx context.Context, sdp **Server) context.Context {
	return context.WithValue(ctx, serverKey{}, sdp)
}

func SetServer(ctx context.Context, s *Server) {
	if sdp, ok := ctx.Value(serverKey{}).(**Server); ok {
		*sdp = s
	}
}
