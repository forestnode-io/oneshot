package signallingserver

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	_ "embed"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

const maxBodySize = 1024 * 1024

//go:generate make webrtc-client
//go:embed main.js
var BrowserClientJS string

//go:embed index.template.html
var HTMLTemplate string

func init() {
	if len(BrowserClientJS) == 0 {
		panic("browserClientJS is empty")
	}
	BrowserClientJS = "<script>\n" + BrowserClientJS + "\n</script>"

	if len(HTMLTemplate) == 0 {
		panic("htmlTemplate is empty")
	}
}

type server struct {
	iceServerURL       string
	htmlClientTemplate *template.Template
	os                 *oneshotServer
	l                  net.Listener
	pendingSessionID   int32
	requiredID         string
	path               string
}

func newServer(iceServerURL string, requiredID string) (*server, error) {
	t, err := template.New("root").Parse(HTMLTemplate)
	return &server{
		iceServerURL:       iceServerURL,
		pendingSessionID:   -1,
		htmlClientTemplate: t,
		requiredID:         requiredID,
	}, err
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
			if err := s.os.Close(); err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Printf("error closing oneshot server: %v", err)
				}
			}
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

	log.Printf("listening for http traffic on %s", httpAddr)

	s.l = l
	for {
		conn, err := s.l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			} else {
				log.Printf("error accepting connection: %v", err)
			}
			return fmt.Errorf("error accepting connection: %w", err)
		}

		if err := s.handleOneshotServer(ctx, conn); err != nil {
			log.Printf("error handling oneshot server: %v", err)
		}
	}
}

// handleOneshotServer handles a new connection to the signalling server.
// If the server is already in use, it will return a BUSY response.
// Otherwise, it will create a new oneshot server.
// handleOneshotServer takes over the connection and will close it when it is done.
func (s *server) handleOneshotServer(ctx context.Context, conn net.Conn) error {
	defer func() {
		log.Printf("closing connection: %v", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Printf("error closing connection: %v", err)
			}
		}
	}()

	var err error
	if s.os != nil {
		if _, err = conn.Write([]byte("BUSY")); err != nil {
			return fmt.Errorf("error writing BUSY response: %w", err)
		}
		return nil
	}

	defer func() {
		s.os = nil
	}()

	if s.os, err = newOneshotServer(ctx, s.requiredID, conn, s.handleURLRequest); err != nil {
		return fmt.Errorf("error creating new oneshot server: %w", err)
	}

	log.Printf("new oneshot server arrived: %v", conn.RemoteAddr())

	<-s.os.Done()

	log.Println("session ended")

	return nil
}

func (s *server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if s.os == nil {
		http.Error(w, "no oneshot server available", http.StatusServiceUnavailable)
		return
	}

	if r.Method == http.MethodGet {
		s.handleGet(w, r)
	} else if r.Method == http.MethodPost {
		s.handlePost(w, r)
	}
}

func (s *server) handleGet(w http.ResponseWriter, r *http.Request) {
	if s.path != "" {
		if r.URL.Path != s.path {
			http.NotFound(w, r)
			return
		}
	}

	if -1 < s.pendingSessionID {
		http.Error(w, "busy", http.StatusServiceUnavailable)
		return
	}

	if s.os.Arrival.BasicAuth != nil {
		// we need to do basic auth
		username, password, ok := r.BasicAuth()
		if ok {
			uHash := sha256.Sum256([]byte(username))
			pHash := sha256.Sum256([]byte(password))

			uMatch := subtle.ConstantTimeCompare(uHash[:], s.os.Arrival.BasicAuth.UsernameHash)
			pMatch := subtle.ConstantTimeCompare(pHash[:], s.os.Arrival.BasicAuth.PasswordHash)

			if uMatch == 0 || pMatch == 0 {
				w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"; charset="UTF-8"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	}

	s.pendingSessionID = rand.Int31()
	offer, err := s.os.RequestOffer(r.Context(), s.pendingSessionID)
	if err != nil {
		log.Printf("error requesting offer: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	offerBytes, err := offer.MarshalJSON()
	if err != nil {
		log.Printf("error marshaling offer: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if accept := r.Header.Get("Accept"); strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		payload, err := json.Marshal(map[string]any{
			"SessionID": s.pendingSessionID,
			"Offer":     string(offer),
		})
		if err != nil {
			log.Printf("error marshaling response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(payload)
		if err != nil {
			log.Printf("error writing response: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	err = s.htmlClientTemplate.Execute(w, map[string]any{
		"AutoConnect":  true,
		"ClientJS":     template.HTML(BrowserClientJS),
		"ICEServerURL": s.iceServerURL,
		"SessionID":    s.pendingSessionID,
		"Offer":        string(offerBytes),
	})
	if err != nil {
		log.Printf("error writing response: %v", err)
	}

	log.Printf("sent offer for session id %d", s.pendingSessionID)
}

func (s *server) handlePost(w http.ResponseWriter, r *http.Request) {
	if s.pendingSessionID < 0 {
		log.Printf("received answer without pending offer")
		http.Error(w, "no pending offer", http.StatusServiceUnavailable)
		return
	}

	log.Printf("received answer for session id %d", s.pendingSessionID)

	defer r.Body.Close()
	lr := io.LimitReader(r.Body, maxBodySize)
	body, err := io.ReadAll(lr)
	if err != nil {
		log.Printf("error reading body: %v", err)
		http.Error(w, "unable to read body", http.StatusInternalServerError)
		return
	}

	var answer struct {
		Answer    string
		SessionID int32
	}

	if err := json.Unmarshal(body, &answer); err != nil {
		log.Printf("error unmarshaling answer: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if answer.SessionID != s.pendingSessionID {
		log.Printf("received answer with invalid id: %d (expected %d)", answer.SessionID, s.pendingSessionID)
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err = s.os.SendAnswer(r.Context(), s.pendingSessionID, sdp.Answer(answer.Answer)); err != nil {
		log.Printf("error sending answer: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("answer sent for session id %d, closing connection to oneshot server", s.pendingSessionID)

	s.pendingSessionID = -1
	if err = s.os.Close(); err != nil {
		log.Printf("error closing oneshot server connection: %v", err)
	}
	s.os = nil
}

func (s *server) handleURLRequest(rurl string, required bool) (string, error) {
	if rurl == "" {
		if required {
			return "", errors.New("no url provided")
		}
		return "", nil
	}

	u, err := url.Parse(rurl)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}

	s.path = u.Path

	return rurl, nil
}
