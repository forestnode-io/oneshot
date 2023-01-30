package http

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
)

const shutdownTimeout = 500 * time.Millisecond

type _wr struct {
	w    http.ResponseWriter
	r    *http.Request
	done func()
}

type Server struct {
	server http.Server

	PreSuccessHandler  http.HandlerFunc
	PostSuccessHandler http.HandlerFunc

	TLSCert, TLSKey string
	Timeout         time.Duration

	queue chan _wr
}

func NewServer(ctx context.Context, preSucc, postSucc http.HandlerFunc, mw ...Middleware) *Server {
	s := Server{
		PreSuccessHandler:  preSucc,
		PostSuccessHandler: postSucc,

		queue: make(chan _wr, runtime.NumCPU()),

		server: http.Server{},
	}
	s.server.BaseContext = func(l net.Listener) context.Context {
		return ctx
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		wr := _wr{
			w: noCacheWriter{ResponseWriter: w},
			r: r,
		}

		// give the handler a way of telling us it's done
		doneChan := make(chan struct{})
		wr.done = func() {
			close(doneChan)
		}

		// give the request to the worker(s)
		s.queue <- wr
		// wait for the worker to tell us it's done
		<-doneChan
	}

	// apply middleware
	for _, mw := range mw {
		handler = mw(handler)
	}

	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(handler)
	s.server.Handler = r

	return &s
}

// Serve starts the server and blocks until the context is canceled or a handler raises a success event.
// Serve waits for all workers to finish before returning.
func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	wg := sync.WaitGroup{}
	shutdownErrChan := make(chan error)

	postSuccWorker := func() {
		for wr := range s.queue {
			s.PostSuccessHandler(wr.w, wr.r)
			wr.done()
		}
		wg.Done()
	}

	preSuccWorker := func() {
		for wr := range s.queue {
			s.PreSuccessHandler(wr.w, wr.r.WithContext(ctx))
			wr.done()

			if events.Succeeded(ctx) {
				cpuCount := runtime.NumCPU()
				wg.Add(cpuCount)
				for i := 0; i < cpuCount; i++ {
					go postSuccWorker()
				}

				ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
				shutdownErrChan <- s.server.Shutdown(ctx)
				cancel()

				return
			}
		}
	}

	go preSuccWorker()
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		shutdownErrChan <- s.server.Shutdown(ctx)
	}()

	if 0 < s.Timeout {
		l = oneshotnet.NewListenerTimer(l, s.Timeout)
	}

	var err error
	if s.TLSCert != "" && s.TLSKey != "" {
		err = s.server.ServeTLS(l, s.TLSCert, s.TLSKey)
	} else {
		err = s.server.Serve(l)
	}
	if err = cleanServerShutdownErr(err); err != nil {
		log.Printf("server error: %v", err)
	}

	// wait for the server to shutdown
	err = <-shutdownErrChan

	close(s.queue)

	// wait for the workers to finish
	wg.Wait()

	return cleanServerShutdownErr(err)
}

func cleanServerShutdownErr(err error) error {
	if errors.Is(err, http.ErrServerClosed) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return nil
	}

	return err
}

type noCacheWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w noCacheWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w noCacheWriter) WriteHeader(code int) {
	w.wroteHeader = true
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.ResponseWriter.WriteHeader(code)
}
