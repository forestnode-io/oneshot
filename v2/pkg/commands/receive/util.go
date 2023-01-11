package receive

import (
	"regexp"
	"strings"
)

var (
	lf   = []byte{10}
	crlf = []byte{13, 10}
)

var regex = regexp.MustCompile(`^?\w?filename="?(.+)"?\w?$?`)

func fileName(s string) string {
	subs := regex.FindStringSubmatch(s)
	if len(subs) > 1 {
		ss := strings.TrimSuffix(subs[1], `"`)
		return strings.TrimSuffix(ss, `;`)
	}
	return ""
}
