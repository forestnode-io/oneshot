package discoveryserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/discovery-server/template"
	"github.com/raphaelreyna/oneshot/v2/pkg/log"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

func (s *server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		log = zerolog.Ctx(r.Context()).With().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Str("host", r.Host).
			Logger()
		ctx = log.WithContext(r.Context())
	)
	r = r.WithContext(ctx)
	log.Info().Msg("got request")
	defer log.Info().Msg("finished handling request")

	r.URL.Scheme = "http"
	r.URL.Host = r.Host

	addrURL := url.URL{
		Scheme: "http",
		Host:   r.Host,
		Path:   r.URL.Path,
	}
	addrString := strings.TrimSuffix(addrURL.String(), "/")

	if s.assignedURL == "" || s.assignedURL != addrString || s.os == nil {
		s.error(w, r, http.StatusNotFound,
			"No pending oneshot found",
			"Please make sure you have a pending oneshot before trying to connect to this server.",
		)
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
	if strings.Contains(accept, "application/json") {
		s.handleAcceptJSON(w, r)
		return
	}

	// default to text/html
	s.handleGET_HTML(w, r)
}

func (s *server) handleGET_HTML(w http.ResponseWriter, r *http.Request) {
	var (
		log    = zerolog.Ctx(r.Context())
		config = s.config.Subcommands.DiscoveryServer
	)

	if s.pendingSessionID != "" || s.os == nil {
		s.error(w, r, http.StatusNotFound,
			"No pending oneshot found",
			"Please make sure you have a pending oneshot before trying to connect to this server.",
		)
		return
	}

	if ba := s.os.Arrival.BasicAuth; ba != nil {
		log.Debug().Msg("checking basic auth")

		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		log.Debug().Msg("request contained basic auth credentials")

		uHash := sha256.Sum256([]byte(user))
		if !bytes.Equal(uHash[:], ba.UsernameHash) {
			w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		log.Debug().Msg("username hash matched")

		if bcrypt.CompareHashAndPassword(ba.PasswordHash, []byte(pass)) != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="oneshot"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		log.Debug().Msg("password hash matched")
	}

	sessionID := uuid.NewString()
	expirationTime := time.Now().Add(10 * time.Second)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_id": sessionID,
		"expires":    expirationTime.Unix(),
	}).SignedString([]byte(config.JWT.Value))
	if err != nil {
		log.Error().Err(err).
			Msg("error signing jwt")

		s.error(w, r, http.StatusInternalServerError,
			"Internal Server Error",
			"An internal server error occurred. Please try again later.",
		)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expirationTime,
	})
	if r.Header.Get("User-Agent") == "oneshot" {
		log.Debug().Msg("oneshot client detected")

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
		log.Error().Err(err).
			Msg("error writing response")
	}
}

func (s *server) handleAcceptJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleAcceptJSON_GET(w, r)
	} else {
		s.handleAcceptJSON_POST(w, r)
	}
}

// handleAcceptJSON_GET handles the GET request from the client asking for
// the offer and rtc config. It queues up the request to be handled by a worker
// so that the oneshot server only has to handle one request at a time.
func (s *server) handleAcceptJSON_GET(w http.ResponseWriter, r *http.Request) {
	var (
		log                = zerolog.Ctx(r.Context())
		sessionTokenString = r.Header.Get("X-Session-Token")
		config             = s.config.Subcommands.DiscoveryServer
	)

	// parse the token string into a token
	token, err := jwt.Parse(sessionTokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT.Value), nil
	})
	if err != nil {
		log.Warn().Err(err).
			Msg("error parsing session token")

		s.error(w, r, http.StatusUnauthorized,
			"Invalid Session Token",
			"Your session token is invalid. Please try again.",
		)
		return
	}

	// verify the token algorithm hasnt been changed
	if token.Method != jwt.SigningMethodHS256 {
		log.Warn().
			Str("alg", fmt.Sprintf("%v", token.Header["alg"])).
			Msg("invalid signing method")

		s.error(w, r, http.StatusUnauthorized,
			"Invalid Session Token",
			"Your session token is invalid. Please try again.",
		)
		return
	}

	// extract the claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Warn().
			Str("type", fmt.Sprintf("%T", token.Claims)).
			Msg("invalid claims type")

		s.error(w, r, http.StatusUnauthorized,
			"Invalid Session Token",
			"Your session token is invalid. Please try again.",
		)
		return
	}

	// get the expiration time from the claims
	expiresIface, ok := claims["expires"]
	if !ok {
		log.Warn().
			Msg("missing expires claim")

		s.error(w, r, http.StatusUnauthorized,
			"Expired Session Token",
			"Your session token is expired. Please try again.",
		)
		return
	}

	// apparently int64 will be marshalled as float64
	expiresUnixFloat, ok := expiresIface.(float64)
	if !ok {
		log.Warn().
			Str("type", fmt.Sprintf("%T", expiresIface)).
			Msg("invalid expires type")

		s.error(w, r, http.StatusUnauthorized,
			"Expired Session Token",
			"Your session token is expired. Please try again.",
		)
		return
	}

	// check if the token has expired
	if expires := time.Unix(int64(expiresUnixFloat), 0); time.Now().After(expires) {
		log.Warn().
			Time("expires", expires).
			Msg("session token expired")

		s.error(w, r, http.StatusUnauthorized,
			"Expired Session Token",
			"Your session token is expired. Please try again.",
		)
		return
	}

	// get the session id from the claims
	sessionIDIface, ok := claims["session_id"]
	if !ok {
		log.Warn().Msg("missing session_id claim")

		s.error(w, r, http.StatusUnauthorized,
			"Invalid Session Token",
			"Your session token is invalid. Please try again.",
		)
		return
	}

	sessionID, ok := sessionIDIface.(string)
	if !ok {
		log.Warn().
			Str("type", fmt.Sprintf("%T", sessionIDIface)).
			Msg("invalid session_id type")

		s.error(w, r, http.StatusUnauthorized,
			"Invalid Session Token",
			"Your session token is invalid. Please try again.",
		)
		return
	}
	if sessionID == "" {
		log.Warn().Msg("session_id claim is empty")

		s.error(w, r, http.StatusUnauthorized,
			"Invalid Session Token",
			"Your session token is invalid. Please try again.",
		)
		return
	}

	done, err := s.queueRequest(sessionID, w, r)
	if err != nil {
		log.Error().Err(err).
			Msg("error queueing request")

		s.error(w, r, http.StatusServiceUnavailable,
			"Client queue is full",
			"Too many clients are currently queued to connect. Please try again later.",
		)
		return
	}
	<-done
}

// handleAcceptJSON_POST handles the POST request from the client that contains the answer to
// the offer provided earlier by the worker.
func (s *server) handleAcceptJSON_POST(w http.ResponseWriter, r *http.Request) {
	log := zerolog.Ctx(r.Context())

	if s.pendingSessionID == "" {
		log.Warn().Msg("received answer without pending session")

		s.error(w, r, http.StatusBadRequest,
			"No pending oneshot found",
			"Please make sure you have a pending oneshot before trying to connect to this server.",
		)
		return
	}

	var req struct {
		Answer    string `json:"answer"`
		SessionID string `json:"sessionID"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.error(w, r, http.StatusBadRequest,
			"Invalid JSON",
			"Please make sure you are sending valid JSON.",
		)
		return
	}
	r.Body.Close()

	if req.SessionID != s.pendingSessionID {
		s.error(w, r, http.StatusBadRequest,
			"Invalid Session ID",
			"Please make sure you are sending the correct session ID.",
		)
		return
	}

	ctx := r.Context()
	if err = s.os.SendAnswer(ctx, req.SessionID, sdp.Answer(req.Answer)); err != nil {
		s.error(w, r, http.StatusInternalServerError,
			"Error sending answer to oneshot server",
			"Please try again later.",
		)
		return
	}
}

var ErrClientQueueFull = errors.New("client queue is full")

// worker handles queued up requests from the client for an offer and rtc config.
// the client asks for this after being sent the html page and running the client script.
func (s *server) worker() {
	log := log.Logger()

	for bundle := range s.queue {
		func() {
			defer close(bundle.done)

			var (
				sessionID = bundle.sessionID
				w         = bundle.w
				r         = bundle.r
			)

			if s.pendingSessionID != "" {
				s.error(w, r, http.StatusConflict,
					"Session already exists",
					"Please try again later.",
				)
				return
			}
			s.pendingSessionID = sessionID

			ctx := r.Context()
			offer, err := s.os.RequestOffer(ctx, sessionID, s.rtcConfig)
			if err != nil {
				log.Error().Err(err).
					Str("session_id", sessionID).
					Msg("error requesting offer from oneshot server")

				s.error(w, r, http.StatusInternalServerError,
					"Error requesting offer from oneshot server",
					"Please try again later.",
				)
				return
			}

			sd, err := offer.WebRTCSessionDescription()
			if err != nil {
				log.Error().Err(err).
					Msg("error getting session description")

				s.error(w, r, http.StatusInternalServerError,
					"Internal Server Error",
					"Please try again later.",
				)
				return
			}

			resp := ClientOfferRequestResponse{
				RTCSessionDescription: &sd,
				SessionID:             sessionID,
				RTCConfiguration:      s.rtcConfig,
			}
			payload, err := json.Marshal(resp)
			if err != nil {
				log.Error().Err(err).
					Msg("error marshaling response")

				s.error(w, r, http.StatusInternalServerError,
					"Internal Server Error",
					"Please try again later.",
				)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(payload); err != nil {
				log.Error().Err(err).
					Msg("error writing response")
			}
		}()
	}
}

func (s *server) error(w http.ResponseWriter, r *http.Request, status int, title, description string) {
	var (
		log    = log.Logger()
		accept = r.Header.Get("Accept")
	)

	switch {
	case strings.Contains(accept, "application/json"):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		payload, err := json.Marshal(map[string]any{
			"error":       title,
			"description": description,
		})
		if err != nil {
			log.Error().Err(err).
				Msg("error marshaling error")
			return
		}
		if _, err = w.Write(payload); err != nil {
			log.Error().Err(err).
				Msg("error writing error")
		}
		return
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		err := template.Error(w, title, description, s.errorPageTitle)
		if err != nil {
			log.Error().Err(err).
				Msg("error writing error")
		}
	}
}
