package http

import (
	"fmt"
	"net/http"
	"strings"
)

func HeaderFromStringSlice(s []string) (http.Header, error) {
	h := make(http.Header)
	for _, hs := range s {
		var (
			parts = strings.SplitN(hs, "=", 2)
			k     = parts[0]
			v     = ""
		)

		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header, must be of the form <HEADER_NAME>=<HEADER_VALUE>: %s", hs)
		}

		v = parts[1]
		var vs = h[k]
		if vs == nil {
			vs = make([]string, 0)
		}
		vs = append(vs, v)
		h[k] = vs
	}

	return h, nil
}
