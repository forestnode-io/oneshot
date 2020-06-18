package handlers

import (
	"errors"
	"log"
	"net/http"
)

var AuthErr = errors.New("unauthenticated")

func Authenticate(username, password string,
	unauthenticated http.HandlerFunc,
	authenticated func(w http.ResponseWriter, r *http.Request) error,
) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		u, p, ok := r.BasicAuth()
		if !ok {
			log.Println("no auth provided")
			unauthenticated(w, r)
			return AuthErr
		}
		// Whichever field is missing is not checked
		if username != "" && username != u {
			log.Printf("expected username: %s got: %s\n", username, u)
			unauthenticated(w, r)
			return AuthErr
		}
		if password != "" && password != p {
			log.Printf("expected password: %s got: %s\n", password, p)
			unauthenticated(w, r)
			return AuthErr
		}
		log.Printf("authenticated\nusername: %s\npassword: %s\nu: %s\np: %s\n",
			username, password, u, p,
		)
		return authenticated(w, r)
	}
}
