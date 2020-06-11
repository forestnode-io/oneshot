package server

import (
	"context"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	router *http.ServeMux
	mutex  *sync.Mutex
	server *http.Server
	timer  *time.Timer
	file   *File
	done   bool

	Port     string
	ErrorLog *log.Logger
	InfoLog  *log.Logger
	Timeout  time.Duration // zero value -> max duration
	Done     chan struct{}
}

func NewServer(file *File) *Server {
	s := &Server{
		router: http.NewServeMux(),
		mutex:  &sync.Mutex{},
		file:   file,
	}
	s.server = &http.Server{Handler: s}

	s.router.HandleFunc("/", s.handleDownload)
	return s
}

func (s *Server) Serve(ctx context.Context) error {
	if s.Timeout == 0 {
		max := math.MaxInt64
		s.Timeout = time.Duration(max)
	}

	s.server.Addr = ":" + s.Port

	s.timer = time.AfterFunc(s.Timeout, func() {
		s.mutex.Lock()
		s.done = true
		s.mutex.Unlock()

		if s.InfoLog != nil {
			duration := s.Timeout.String()
			s.InfoLog.Printf("no client requested the file after %s; timing out ...\n", duration)
		}
		s.Stop(ctx)
	})

	if s.InfoLog != nil {
		s.InfoLog.Printf("server started; listening on port %s", s.Port)
	}
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.timer == nil {
		return nil
	}
	s.timer.Stop()
	err := s.server.Shutdown(ctx)
	s.Done <- struct{}{}
	return err
}

func (s *Server) Close() error {
	if s.timer == nil {
		return nil
	}
	s.timer.Stop()
	err := s.server.Close()
	s.Done <- struct{}{}
	return err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
