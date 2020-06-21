package server

import (
	"context"
	"errors"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

var OKDoneErr = errors.New("route done")

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

func (s *Server) AddRoute(route *Route) {
	if s.wg == nil {
		s.wg = &sync.WaitGroup{}
		s.wg.Add(1)
		go func() {
			s.wg.Wait()
			s.Done <- s.finishedRoutes
			close(s.Done)
		}()
	}

	okMetric := true
	if route.MaxRequests != 0 {
		okMetric = false
	} else if route.MaxOK == 0 {
		route.MaxOK = 1
	}

	rr := s.router.HandleFunc(route.Pattern, func(w http.ResponseWriter, r *http.Request) {
		var rc int64
		var err error
		route.Lock()
		route.reqCount++

		if okMetric {
			switch {
			case route.okCount >= route.MaxOK:
				route.DoneHandlerFunc(w, r)
			case route.okCount < route.MaxOK:
				if s.InfoLog != nil {
					s.InfoLog.Printf("client connected: %s\n", r.RemoteAddr)
				}
				err = route.HandlerFunc(w, r)

				if err == nil {
					route.okCount++
				} else if s.ErrorLog != nil {
					s.ErrorLog.Println(err)
				}

				if route.okCount == route.MaxOK {
					if err == nil {
						err = OKDoneErr
					}
					s.Lock()
					s.finishedRoutes[route] = err
					s.Unlock()
					s.wg.Done()
				}
			}
			route.Unlock()
			return
		}

		rc = route.reqCount
		route.Unlock()
		switch {
		case rc > route.MaxRequests:
			route.DoneHandlerFunc(w, r)
		case rc <= route.MaxRequests:
			if s.InfoLog != nil {
				s.InfoLog.Printf("client connected: %s\n", r.RemoteAddr)
			}
			err = route.HandlerFunc(w, r)

			if err == nil {
				route.Lock()
				route.okCount++
				route.Unlock()
			} else if s.ErrorLog != nil {
				s.ErrorLog.Println(err)
			}

			if rc == route.MaxRequests {
				if err == nil {
					err = OKDoneErr
				}
				s.Lock()
				s.finishedRoutes[route] = err
				s.Unlock()
				s.wg.Done()
			}
		}
	})

	if len(route.Methods) > 0 {
		rr.Methods(route.Methods...)
	}
}

func (s *Server) Serve() error {
	s.server.Addr = ":" + s.Port

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
		ips, err := s.getHostIPs(true)
		if err != nil {
			s.internalError(err.Error())
			return err
		}
		msg := "listening:\n"
		for _, ip := range ips {
			msg += "\t" + ip + "\n"
		}
		s.infoLog(msg)
		s.serving = true
		return s.server.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}

	ips, err := s.getHostIPs(false)
	if err != nil {
		s.internalError(err.Error())
		return err
	}
	msg := "listening:\n"
	for _, ip := range ips {
		msg += "\t - " + ip + "\n"
	}
	s.infoLog(msg)
	s.serving = true
	return s.server.ListenAndServe()
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

func (s *Server) getHostIPs(tls bool) ([]string, error) {
	ips := []string{}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var home string
	for _, addr := range addrs {
		saddr := addr.String()

		if strings.Contains(saddr, "::") {
			continue
		}

		parts := strings.Split(saddr, "/")
		ip := parts[0] + ":" + s.Port

		if tls {
			ip = "https://" + ip
		} else {
			ip = "http://" + ip
		}

		// Remove localhost since whats the point in sharing with yourself? (usually)
		if parts[0] == "127.0.0.1" || parts[0] == "localhost" {
			home = ip
			continue
		}

		ips = append(ips, ip)
	}

	if len(ips) == 0 {
		ips = append(ips, home)
	}

	return ips, nil
}
