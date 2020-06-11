package server

import (
	"net/http"
	"log"
	"sync"
	"time"
	"math"
)

type Server struct {
	router *http.ServeMux
	mutex *sync.Mutex
	server *http.Server
	timer *time.Timer
	done bool

	Port string
	ErrorLog *log.Logger
	InfoLog *log.Logger
	FilePath string
	Timeout time.Duration // zero value -> max duration
	Done chan struct{}
}

func NewServer() *Server {
	s := &Server{
		router: http.NewServeMux(),
		mutex: &sync.Mutex{},
	}
	s.server = &http.Server{Handler: s}

	s.router.HandleFunc("/", s.handleDownload)
	return s
}

func (s *Server) Serve() error {
	if s.Timeout == 0 {
		max := math.MaxInt64
		s.Timeout = time.Duration(max)
	}

	s.server.Addr = ":" + s.Port

	s.timer = time.AfterFunc(s.Timeout, func(){
		s.mutex.Lock()
		s.done = true
		s.mutex.Unlock()

		if s.InfoLog != nil {
			duration := s.Timeout.String()
			s.InfoLog.Printf("no client requested the file after %s; timing out ...\n", duration)
		}
		s.server.Close()
		s.Done <- struct{}{}
	})

	if s.InfoLog != nil {
		s.InfoLog.Printf("server started; listening on port %s", s.Port)
	}
	return s.server.ListenAndServe()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
