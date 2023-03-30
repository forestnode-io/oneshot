package signallingserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"

	_ "embed"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
	"google.golang.org/grpc"
)

type requestBundle struct {
	w         http.ResponseWriter
	r         *http.Request
	sessionID string
	done      chan struct{}
}

type server struct {
	os         *oneshotServer
	requiredID string
	sessionURL string

	domain string
	scheme string

	pendingSessionID string
	assignedURL      string

	rtcConfig *webrtc.Configuration

	jwtSecret []byte

	queue chan requestBundle
	mu    sync.Mutex

	proto.UnimplementedSignallingServerServer
}

func newServer(requiredID string, config *webrtc.Configuration) *server {
	return &server{
		requiredID: requiredID,
		queue:      make(chan requestBundle, 10),
		rtcConfig:  config,
	}
}

func (s *server) run(ctx context.Context, signallingAddr, httpAddr string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	l, err := net.Listen("tcp", signallingAddr)
	if err != nil {
		return err
	}

	log.Printf("listening for signalling traffic on %s", signallingAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTP)
	hs := http.Server{
		Addr:    httpAddr,
		Handler: mux,
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
		log.Println("shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()

		if s.os != nil {
			s.os.Close()
		}

		if err := hs.Shutdown(ctx); err != nil {
			log.Printf("error shutting down http server: %v", err)
		}

		log.Println("http service shutdown")

		if err := l.Close(); err != nil {
			log.Printf("error closing listener: %v", err)
		}

		log.Println("api service shutdown")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := hs.ListenAndServe(); err != nil {
			cancel()
			if err != http.ErrServerClosed {
				log.Printf("error serving http: %v", err)
			}
		}
	}()

	go s.worker()

	log.Printf("listening for http traffic on %s", httpAddr)
	server := grpc.NewServer()
	proto.RegisterSignallingServerServer(server, s)
	if err := server.Serve(l); err != nil {
		if !errors.Is(err, net.ErrClosed) {
			return err
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
	if rurl == "" {
		if required {
			return "", errors.New("no url provided")
		}
		s.assignedURL = "http://localhost:8080/"
		return s.assignedURL, nil
	}

	u, err := url.Parse(rurl)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	u.Scheme = s.scheme
	u.Host = s.domain
	if u.String() != rurl && required {
		return "", nil
	}

	s.assignedURL = u.String()

	return rurl, nil
}

func (s *server) Connect(stream proto.SignallingServer_ConnectServer) error {
	log.Printf("new connection")
	if s.os != nil {
		log.Printf("already connected")
		return errors.New("already connected")
	}

	var (
		ctx          = stream.Context()
		resetPending = func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.pendingSessionID = ""
		}
		err error
	)
	s.os, err = newOneshotServer(ctx, s.requiredID, stream, resetPending, s.handleURLRequest)
	if err != nil {
		log.Printf("error creating oneshot server: %v", err)
		return err
	}
	log.Printf("oneshot server created")

	// hold the stream open until the oneshot server is done.
	// from this point on, the http server will be the only thing
	// using the stream, we just need to hold it open here.
	log.Printf("waiting for oneshot server to finish")
	select {
	case <-ctx.Done():
		log.Printf("context done")
	case <-s.os.done:
		log.Printf("oneshot server finished")
	}
	s.os = nil
	s.pendingSessionID = ""

	return nil
}
