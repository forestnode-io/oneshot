package server

import (
	"net/http"
)

func (s *Server) authenticate(downloader http.HandlerFunc) http.HandlerFunc {
	unauthorized := func(w http.ResponseWriter) {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authenticating {
			username, password, ok := r.BasicAuth()
			if !ok {
				unauthorized(w)
				return
			}
			// Whichever field is missing is not checked
			if s.Username != "" && s.Username != username {
				unauthorized(w)
				return
			}
			if s.Password != "" && s.Password != password {
				unauthorized(w)
				return
			}
		}
		downloader(w, r)
	}
}
