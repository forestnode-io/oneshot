package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

var TimeoutErr = errors.New("server timed out")

type Server struct {
	Port string

	// Username to use for authentication.
	// If not the empty string, authentication headers will be set.
	// If Username is set but Password is not, then the client may enter any password.
	Username string
	// Password to use for authentication.
	// If not the empty string, authentication headers will be set.
	// If Password is set but Username is not, then the client may enter any username.
	Password string

	// Certfile is the public certificate that should be used for TLS
	CertFile string
	// Keyfile is the private key that should be used for TLS
	KeyFile string

	// Timeout sets how long we should wait for the client for before server shutdown.
	Timeout time.Duration // zero value -> no timeout / infinite duration
	// Download sets if a download should be triggered on the clients browser.
	// This is done by setting an appropriate "Content-Disposition" header.
	Download bool

	// Done signals when the server has shutdown regardless of value.
	// If nil, the file was sent; otherwise, value contains an error describing why the file could not be sent.
	Done chan error

	ErrorLog *log.Logger
	InfoLog  *log.Logger

	router         *http.ServeMux
	server         *http.Server
	timer          *time.Timer
	file           *File
	err            error
	authenticating bool

	serving bool // Set to true after Serve() is called, false after Stop() or Close()
}

func NewServer(file *File) *Server {
	s := &Server{
		router: http.NewServeMux(),
		file:   file,
	}
	s.server = &http.Server{Handler: s}

	s.router.HandleFunc("/", s.authenticate(s.handleDownload(s.file)))
	return s
}

func (s *Server) Serve(ctx context.Context) error {
	s.server.Addr = ":" + s.Port
	if s.Username != "" || s.Password != "" {
		s.authenticating = true
	}

	if s.CertFile != "" && s.KeyFile != "" {
		switch {
		case s.CertFile == "":
			err := errors.New("given cert file for HTTPS but no key file. exit")
			s.internalError(err.Error())
			return err
		case s.KeyFile == "":
			err := errors.New("given key file for HTTPS but no cert file. exit")
			s.internalError(err.Error())
			return err
		}
		s.infoLog("HTTPS server started; listening on port %s", s.Port)
		s.startTimer(ctx)
		s.serving = true
		return s.server.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}

	s.infoLog("HTTP server started; listening on port %s", s.Port)
	s.startTimer(ctx)
	s.serving = true
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if !s.serving {
		return nil
	}

	if s.timer != nil {
		s.timer.Stop()
	}

	err := s.server.Shutdown(ctx)
	s.Done <- s.err
	return err
}

func (s *Server) Close() error {
	if !s.serving {
		return nil
	}

	if s.timer != nil {
		s.timer.Stop()
	}

	err := s.server.Close()
	s.Done <- s.err
	return err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) startTimer(ctx context.Context) {
	if s.Timeout != 0 {
		s.infoLog("server timeout in %v\n", s.Timeout)
		s.timer = time.AfterFunc(s.Timeout, func() {
			s.file.Lock()

			s.file.Requested()
			s.err = TimeoutErr

			s.file.Unlock()

			s.Stop(ctx)
		})
	}
}
