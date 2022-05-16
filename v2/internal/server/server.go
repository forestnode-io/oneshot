package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/raphaelreyna/oneshot/v2/internal/summary"
)

type connKey struct{}

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request) (*summary.Request, error)
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
	requestsWG    sync.WaitGroup

	summary     *summary.Summary
	succChan    chan struct{}
	stopWorkers func()

	handler Handler
}

func NewServer(handler Handler) *Server {
	s := Server{
		requestsQueue: make(chan _wr),
		handler:       handler,
		succChan:      make(chan struct{}),
		summary:       summary.NewSummary(time.Now()),
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

	go func() {
		<-s.succChan
		s.requestsWG.Wait()
		errs <- nil
	}()

	err := <-errs
	if err != nil {
		return err
	}

	s.summary.End()
	s.Shutdown(ctx)
	je := json.NewEncoder(os.Stdout)
	je.SetIndent("", "\t")
	je.Encode(s.summary)
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.requestsQueue)
	s.stopWorkers()

	return s.Server.Shutdown(ctx)
}

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	// go dark if the transfer succeeded
	if s.summary.Succesful() {
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

	s.requestsWG.Add(1)
	s.requestsQueue <- wr

	<-done
	s.requestsWG.Done()
}

func (s *Server) preTransferWorker(ctx context.Context) {
	for {
		if s.summary.Succesful() {
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

			smry, err := s.handler.ServeHTTP(wr.w, wr.r)
			if err != nil {
				s.summary.AddFailure(smry)
			} else {
				s.summary.SucceededWith(smry)
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
