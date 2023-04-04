package discoveryserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	_ "embed"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/log"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var kaep = keepalive.EnforcementPolicy{
	MinTime: 3 * time.Second, // If a client pings more than once every 3 seconds, terminate the connection
}

var kasp = keepalive.ServerParameters{
	MaxConnectionIdle:     9 * time.Second, // If a client is idle for 6 seconds, send a GOAWAY
	MaxConnectionAgeGrace: 1 * time.Second, // Allow 1 seconds for pending RPCs to complete before forcibly closing connections
	Time:                  3 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
	Timeout:               1 * time.Second, // Wait 1 second for the ping ack before assuming the connection is dead
}

type requestBundle struct {
	w         http.ResponseWriter
	r         *http.Request
	sessionID string
	done      chan struct{}
}

type server struct {
	os *oneshotServer

	pendingSessionID string
	assignedURL      string

	rtcConfig *webrtc.Configuration
	config    Config

	errorPageTitle string

	queue chan requestBundle
	mu    sync.Mutex

	proto.UnimplementedSignallingServerServer
}

func newServer(c *Config) (*server, error) {
	if err := c.SetDefaults(); err != nil {
		return nil, fmt.Errorf("unable to set defaults: %w", err)
	}

	// read jwt secret from file if necessary
	if c.JWTSecretConfig.Value == "" {
		data, err := os.ReadFile(c.JWTSecretConfig.Path)
		if err != nil {
			return nil, fmt.Errorf("unable to readt JWT secret from file")
		}
		c.JWTSecretConfig.Value = string(data)
	}
	if c.JWTSecretConfig.Value == "" {
		return nil, fmt.Errorf("JWT secret is empty")
	}

	rc, err := c.WebRTCConfiguration.WebRTCConfiguration()
	if err != nil {
		return nil, fmt.Errorf("unable to create webrtc configuration: %w", err)
	}

	s := server{
		queue:     make(chan requestBundle, c.MaxClientQueueSize),
		rtcConfig: rc,
		config:    *c,
	}

	return &s, nil
}

func (s *server) run(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		l   net.Listener
		err error

		dc = s.config.Servers.Discovery
		hc = s.config.Servers.HTTP
	)

	if tc := s.config.Servers.Discovery.TLS; tc != nil {
		cert, err := tls.LoadX509KeyPair(tc.Cert, tc.Key)
		if err != nil {
			return fmt.Errorf("unable to load tls key pair: %w", err)
		}
		l, err = tls.Listen("tcp", dc.Addr, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		if err != nil {
			return fmt.Errorf("unable to listen for tls traffic on %s: %w", dc.Addr, err)
		}
	} else {
		l, err = net.Listen("tcp", dc.Addr)
		if err != nil {
			return fmt.Errorf("unable to listen for traffic on %s: %w", dc.Addr, err)
		}
	}

	log.Info().
		Str("addr", dc.Addr).
		Msg("listening for api traffic")

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTP)
	hs := http.Server{
		Addr:    hc.Addr,
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	defer func() {
		wg.Wait()
		events.Stop(ctx)
	}()

	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Info().Msg("shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()

		if s.os != nil {
			s.os.Close()
		}

		if err := hs.Shutdown(ctx); err != nil {
			log.Error().Err(err).
				Msg("error shutting down http server")
		}

		log.Info().Msg("http service shutdown")

		if err := l.Close(); err != nil {
			log.Error().Err(err).
				Msg("error closing listener")
		}

		log.Info().Msg("discovery service shutdown")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if hc.TLS != nil {
			if err := hs.ListenAndServeTLS(hc.TLS.Cert, hc.TLS.Key); err != nil {
				cancel()
				if err != http.ErrServerClosed {
					log.Error().Err(err).
						Msg("error serving http")
				}
			}
			return
		} else {
			if err := hs.ListenAndServe(); err != nil {
				cancel()
				if err != http.ErrServerClosed {
					log.Error().Err(err).
						Msg("error serving http")
				}
			}
		}
	}()

	go s.worker()

	log.Info().
		Str("addr", hc.Addr).
		Msg("listening for http traffic")
	server := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
	)
	proto.RegisterSignallingServerServer(server, s)
	if err := server.Serve(l); err != nil {
		if !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("error serving grpc: %w", err)
		}
	}

	return nil
}

func (s *server) queueRequest(sessionID string, w http.ResponseWriter, r *http.Request) (<-chan struct{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		currentClientQueueCount = len(s.queue)
		maxClientQueueCount     = cap(s.queue)
	)
	if maxClientQueueCount <= currentClientQueueCount {
		return nil, ErrClientQueueFull
	}

	done := make(chan struct{})
	s.queue <- requestBundle{
		w:         w,
		r:         r,
		sessionID: sessionID,
		done:      done,
	}

	return done, nil
}

func (s *server) handleURLRequest(rurl string, required bool) (string, error) {
	var (
		scheme = s.config.URLAssignment.Scheme
		domain = s.config.URLAssignment.Domain + fmt.Sprintf(":%d", s.config.URLAssignment.Port)
	)

	if rurl == "" {
		if required {
			return "", errors.New("no url provided")
		}
		u := url.URL{
			Scheme: scheme,
			Host:   domain,
		}
		s.assignedURL = u.String()
		return s.assignedURL, nil
	}

	u, err := url.Parse(rurl)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	u.Scheme = scheme
	u.Host = domain
	if u.String() != rurl && required {
		return "", nil
	}

	s.assignedURL = u.String()

	return rurl, nil
}

func (s *server) Connect(stream proto.SignallingServer_ConnectServer) error {
	var (
		log = log.Logger()
		ctx = log.WithContext(stream.Context())
	)

	log.Debug().Msg("new connection")

	if s.os != nil {
		log.Debug().
			Msg("got connection while another is in progress")

		return errors.New("already connected")
	}

	var (
		resetPending = func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.pendingSessionID = ""
		}
		err error
	)
	s.os, err = newOneshotServer(ctx, s.config.RequiredID.Value, stream, resetPending, s.handleURLRequest)
	if err != nil {
		log.Error().Err(err).
			Msg("error creating oneshot server")
		return err
	}
	log.Debug().Msg("new oneshot server arrival")

	// hold the stream open until the oneshot server is done.
	// from this point on, the http server will be the only thing
	// using the stream, we just need to hold it open here.
	select {
	case <-ctx.Done():
	case <-s.os.done:
	}
	log.Debug().Msg("oneshot server disconnected")
	s.os = nil
	s.pendingSessionID = ""

	return nil
}
