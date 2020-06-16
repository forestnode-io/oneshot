package server

import (
	"context"
	"errors"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	router         *http.ServeMux
	mutex          *sync.Mutex
	server         *http.Server
	timer          *time.Timer
	file           *File
	done           bool
	authenticating bool

	Port     string
	Username string
	Password string
	CertFile string
	KeyFile  string
	ErrorLog *log.Logger
	InfoLog  *log.Logger
	Timeout  time.Duration // zero value -> max duration
	Download bool          // should "Content-Disposition" header be set?
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
		max := math.MaxInt32
		s.Timeout = time.Duration(max)
	}

	s.server.Addr = ":" + s.Port

	if s.CertFile != "" && s.KeyFile != "" {
		if s.InfoLog != nil {
			s.InfoLog.Printf("HTTPS server started; listening on port %s", s.Port)
		}
		return s.server.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	if s.CertFile != "" {
		err := errors.New("given cert file for HTTPS but no key file. exit\n")
		s.ErrorLog.Printf(err.Error())
		return err
	}
	if s.KeyFile != "" {
		err := errors.New("given key file for HTTPS but no cert file. exit\n")
		s.ErrorLog.Printf(err.Error())
		return err
	}
	if s.InfoLog != nil {
		s.InfoLog.Printf("HTTP server started; listening on port %s", s.Port)
	}

	if s.Username != "" || s.Password != "" {
		s.authenticating = true
	}

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
