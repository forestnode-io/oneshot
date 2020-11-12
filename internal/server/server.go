package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

var OKDoneErr = errors.New("route done")
var OKNotDoneErr = errors.New("not done")

type Server struct {
	Port string

	// Certfile is the public certificate that should be used for TLS
	CertFile string
	// Keyfile is the private key that should be used for TLS
	KeyFile string

	// Done signals when the server has shutdown regardless of value.
	// Each route that finished will have an error message in the map.
	// Routes that finish successfully will have an OKDoneErr error.
	Done chan map[*Route]error

	ErrorLog *log.Logger
	InfoLog  *log.Logger

	HostAddresses []string

	router *mux.Router
	server *http.Server

	serving bool // Set to true after Serve() is called, false after Stop() or Close()

	wg *sync.WaitGroup
	sync.Mutex

	finishedRoutes map[*Route]error
}

func NewServer() *Server {
	s := &Server{
		router:         mux.NewRouter(),
		finishedRoutes: make(map[*Route]error),
	}
	s.server = &http.Server{Handler: s}
	return s
}

func (s *Server) Serve() error {
	s.server.Addr = ":" + s.Port

	scheme := "http://"
	listenAndServe := func() error {
		return s.server.ListenAndServe()
	}

	// Are we using HTTPS?
	if s.CertFile != "" || s.KeyFile != "" {
		// Error out if only cert or key is given
		var err error
		switch {
		case s.CertFile == "":
			err = errors.New("given cert file for HTTPS but no key file. exit")
		case s.KeyFile == "":
			err = errors.New("given key file for HTTPS but no cert file. exit")
		}
		if err != nil {
			s.internalError(err.Error())
			return err
		}

		scheme = "https://"
		listenAndServe = func() error {
			return s.server.ListenAndServeTLS(s.CertFile, s.KeyFile)
		}
	}

	var addresses string
	for _, ip := range s.HostAddresses {
		addresses += "\t- " + scheme + ip + "\n"
	}

	s.infoLog("listening at:\n" + addresses)
	s.serving = true
	return listenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if !s.serving {
		return nil
	}

	return s.server.Shutdown(ctx)
}

func (s *Server) Close() error {
	if !s.serving {
		return nil
	}

	err := s.server.Close()
	return err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.serving {
		s.serving = true
	}
	s.router.ServeHTTP(w, r)
}

func (s *Server) internalError(format string, v ...interface{}) {
	if s.ErrorLog != nil {
		s.ErrorLog.Printf(format, v...)
	}
}

func (s *Server) infoLog(format string, v ...interface{}) {
	if s.InfoLog != nil {
		s.InfoLog.Printf(format, v...)
	}
}
