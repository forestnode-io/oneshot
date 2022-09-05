package out

import (
	"net/url"
)

type URL struct {
	Scheme   string              `json:",omitempty"`
	User     string              `json:",omitempty"`
	Host     string              `json:",omitempty"`
	Path     string              `json:",omitempty"`
	Fragment string              `json:",omitempty"`
	Query    map[string][]string `json:",omitempty"`
}

func newURL(u *url.URL) URL {
	return URL{
		Scheme:   u.Scheme,
		User:     u.User.String(),
		Host:     u.Host,
		Path:     u.Path,
		Fragment: u.Fragment,
		Query:    u.Query(),
	}
}
