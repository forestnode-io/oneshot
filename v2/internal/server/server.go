package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/raphaelreyna/oneshot/v2/internal/events"
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

	success     bool
	succChan    chan struct{}
	timeoutChan chan struct{}

	serveHTTP        http.HandlerFunc
	serveExpiredHTTP http.HandlerFunc

	TLSCert, TLSKey string

	bufferRequestBody bool
	timeout           time.Duration
}

func NewServer(serve, serveExpired http.HandlerFunc) *Server {
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

	go s.preTransferWorker(ctx)

	if 0 < s.timeout {
		s.timeoutChan = make(chan struct{}, 1)
		l = withTimeout(s.timeout, s.timeoutChan, l)
	}

	var err error
	if s.TLSKey != "" {
		err = s.Server.ServeTLS(l, s.TLSCert, s.TLSKey)
	} else {
		err = s.Server.Serve(l)
	}
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			if err := events.GetCancellationError(ctx); err != nil {
				// TODO(raphaelreyna) improve this
				panic(err)
			}
		}
	}

	return nil
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

	s.requestsQueue <- wr
	<-done
}

func (s *Server) preTransferWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(s.requestsQueue)
			s.Server.Shutdown(ctx)
			return
		case wr, ok := <-s.requestsQueue:
			if !ok {
				return
			}

			s.serveHTTP(wr.w, wr.r) // run the command server
			wr.done()
		}
	}
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
