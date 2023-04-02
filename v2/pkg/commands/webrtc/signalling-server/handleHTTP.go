package signallingserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/signalling-server/template"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"golang.org/x/crypto/bcrypt"
)

func (s *server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("http: %s %s", r.Method, r.URL.String())
	log.Printf("host: %s", r.Host)
	r.URL.Scheme = "http"
	r.URL.Host = r.Host

	addrURL := url.URL{
		Scheme: "http",
		Host:   r.Host,
		Path:   r.URL.Path,
	}
	addrString := strings.TrimSuffix(addrURL.String(), "/")
	log.Printf("url: %s", addrString)

	log.Printf("assigned url: %s", s.assignedURL)
	if s.assignedURL == "" || s.assignedURL != addrString || s.os == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if s.os.Arrival.Redirect != "" {
		if s.os.Arrival.RedirectOnly {
			http.Redirect(w, r, s.os.Arrival.Redirect, http.StatusSeeOther)
			return
		}

		if r.URL.Query().Get("x-oneshot-discovery-redirect") != "" {
			http.Redirect(w, r, s.os.Arrival.Redirect, http.StatusSeeOther)
			return
		}
	}

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") {
		log.Printf("accept: text/html")
		s.handleGET_HTML(w, r)
	} else if strings.Contains(accept, "application/json") {
		log.Printf("accept: application/json")
		s.handleAcceptJSON(w, r)
	} else {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
	}
}

func (s *server) handleGET_HTML(w http.ResponseWriter, r *http.Request) {
	if s.pendingSessionID != "" || s.os == nil {
		http.NotFound(w, r)
		return
	}

	if ba := s.os.Arrival.BasicAuth; ba != nil {
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		uHash := sha256.Sum256([]byte(user))
		if !bytes.Equal(uHash[:], ba.UsernameHash) {
			w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if bcrypt.CompareHashAndPassword(ba.PasswordHash, []byte(pass)) != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	sessionID := uuid.NewString()
	expirationTime := time.Now().Add(10 * time.Second)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_id": sessionID,
		"expires":    expirationTime.Unix(),
	}).SignedString([]byte(s.config.JWTSecretConfig.Value))
	if err != nil {
		log.Printf("error signing jwt: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expirationTime,
	})
	if r.Header.Get("User-Agent") == "oneshot" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	tmpltCtx := template.Context{
		AutoConnect: true,
		ClientJS:    template.ClientJS,
		PolyfillJS:  template.PolyfillJS,
	}

	err = template.WriteTo(w, tmpltCtx)
	if err != nil {
		log.Printf("error writing response: %v", err)
	}
}

func (s *server) handleAcceptJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		log.Printf("accept json get")
		s.handleAcceptJSON_GET(w, r)
	} else {
		log.Printf("accept json post")
		s.handleAcceptJSON_POST(w, r)
	}
}

// handleAcceptJSON_GET handles the GET request from the client asking for
// the offer and rtc config. It queues up the request to be handled by a worker
// so that the oneshot server only has to handle one request at a time.
func (s *server) handleAcceptJSON_GET(w http.ResponseWriter, r *http.Request) {
	// grab the raw token string
	sessionTokenString := r.Header.Get("X-Session-Token")

	// parse the token string into a token
	token, err := jwt.Parse(sessionTokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecretConfig.Value), nil
	})
	if err != nil {
		log.Printf("error parsing session token: %v", err)
		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}

	// verify the token algorithm hasnt been changed
	if token.Method != jwt.SigningMethodHS256 {
		log.Printf("invalid signing method: %v", token.Header["alg"])

		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}

	// extract the claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("invalid claims type: %T", token.Claims)

		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}

	// get the expiration time from the claims
	expiresIface, ok := claims["expires"]
	if !ok {
		log.Printf("missing expires claim")

		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}

	// apparently int64 will be marshalled as float64
	expiresUnixFloat, ok := expiresIface.(float64)
	if !ok {
		log.Printf("expires claim is unexpected type: %T", expiresIface)

		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}

	// check if the token has expired
	if expires := time.Unix(int64(expiresUnixFloat), 0); time.Now().After(expires) {
		log.Printf("session token expired")

		http.Error(w, "session token expired", http.StatusUnauthorized)
		return
	}

	// get the session id from the claims
	sessionIDIface, ok := claims["session_id"]
	if !ok {
		log.Printf("missing session_id claim")

		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}

	sessionID, ok := sessionIDIface.(string)
	if !ok {
		log.Printf("session_id claim is unexpected type: %T", sessionIDIface)

		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}
	if sessionID == "" {
		log.Printf("session_id claim is empty")

		http.Error(w, "invalid session token", http.StatusUnauthorized)
		return
	}

	done, err := s.queueRequest(sessionID, w, r)
	if err != nil {
		log.Printf("error queuing request: %v", err)
		http.Error(w, "server is busy", http.StatusServiceUnavailable)
		return
	}
	<-done
}

// handleAcceptJSON_POST handles the POST request from the client that contains the answer to
// the offer provided earlier by the worker.
func (s *server) handleAcceptJSON_POST(w http.ResponseWriter, r *http.Request) {
	if s.pendingSessionID == "" {
		log.Printf("received answer without pending session")

		http.Error(w, "no pending session", http.StatusBadRequest)
		return
	}

	var req struct {
		Answer    string `json:"answer"`
		SessionID string `json:"sessionID"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()

	if req.SessionID != s.pendingSessionID {
		http.Error(w, "invalid sessionID", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	if err = s.os.SendAnswer(ctx, req.SessionID, sdp.Answer(req.Answer)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var ErrClientQueueFull = errors.New("client queue is full")

// worker handles queued up requests from the client for an offer and rtc config.
// the client asks for this after being sent the html page and running the client script.
func (s *server) worker() {
	for bundle := range s.queue {
		func() {
			defer close(bundle.done)

			var (
				sessionID = bundle.sessionID
				w         = bundle.w
				r         = bundle.r
			)

			if s.pendingSessionID != "" {
				http.Error(w, "session already exists", http.StatusConflict)
				return
			}
			s.pendingSessionID = sessionID

			ctx := r.Context()
			offer, err := s.os.RequestOffer(ctx, sessionID, s.rtcConfig)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			sd, err := offer.WebRTCSessionDescription()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// if we have basic auth, include the token in the session description
			if ba := s.os.Arrival.BasicAuth; ba != nil {
				if ba.Token != "" {
					ssd, err := sd.Unmarshal()
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					ssd = ssd.WithValueAttribute("BasicAuthToken", ba.Token)
					ssdBytes, err := ssd.Marshal()
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					sd.SDP = string(ssdBytes)
				}

			}

			resp := ClientOfferRequestResponse{
				RTCSessionDescription: &sd,
				SessionID:             sessionID,
				RTCConfiguration:      s.rtcConfig,
			}
			payload, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(payload); err != nil {
				log.Printf("error writing response: %v", err)
			}
		}()
	}
}
