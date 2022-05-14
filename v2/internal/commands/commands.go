package commands

import (
	"net/http"
	"strings"
)

func headerFromStringSlice(s []string) http.Header {
	h := make(http.Header)
	if s == nil {
		return h
	}

	for _, hs := range s {
		var (
			parts = strings.SplitN(hs, "=", 1)
			k     = parts[0]
			v     = ""
		)

		if len(parts) == 2 {
			v = parts[1]
		}

		var vs = h[k]
		if vs == nil {
			vs = make([]string, 0)
		}
		vs = append(vs, v)
		h[k] = vs
	}

	return h
}
