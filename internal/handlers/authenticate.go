package handlers

import (
	"fmt"
	"net/http"
	srvr "github.com/oneshot-uno/oneshot/internal/server"
)

func Authenticate(username, password string, unauthenticated http.HandlerFunc, authenticated srvr.FailableHandler) srvr.FailableHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		u, p, ok := r.BasicAuth()
		if !ok {
			unauthenticated(w, r)
			return fmt.Errorf("%s connected without providing username and password", r.RemoteAddr)
		}
		// Whichever field is missing is not checked
		if username != "" && username != u {
			unauthenticated(w, r)
			return fmt.Errorf("%s connected with invalid username and password", r.RemoteAddr)
		}
		if password != "" && password != p {
			unauthenticated(w, r)
			return fmt.Errorf("%s connected with invalid username and password", r.RemoteAddr)
		}
		return authenticated(w, r)
	}
}
