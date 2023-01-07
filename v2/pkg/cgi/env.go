package cgi

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
)

func NewEnv(base, inherit []string) []string {
	var (
		envMap = map[string]string{
			"SERVER_SOFTWARE": "oneshot",
		}
		osDefInherit = osDefaultInheritEnv[runtime.GOOS]
	)

	envPath := os.Getenv("PATH")
	if envPath == "" {
		envPath = "/bin:/usr/bin:/usr/ucb:/usr/bsd:/usr/local/bin"
	}
	envMap["PATH"] = envPath

	for _, k := range append(osDefInherit, inherit...) {
		if v := os.Getenv(k); v != "" {
			envMap[k] = v
		}
	}
	for _, e := range base {
		var (
			parts = strings.SplitN(e, "=", 2)
			l     = len(parts)
			k     = parts[0]
			v     = ""
		)
		if l == 2 {
			v = parts[1]
		}
		envMap[k] = v
	}

	var (
		env = make([]string, len(envMap))
		idx int
	)
	for k, v := range envMap {
		env[idx] = k + "=" + v
		idx++
	}

	return removeLeadingDuplicates(env)
}

func AddRequest(env []string, r *http.Request) []string {
	var newEnv = make([]string, len(env))
	copy(newEnv, env)

	if remoteIP, remotePort, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		newEnv = append(env, "REMOTE_ADDR="+remoteIP, "REMOTE_HOST="+remoteIP, "REMOTE_PORT="+remotePort)
	} else {
		newEnv = append(newEnv, "REMOTE_ADDR="+r.RemoteAddr, "REMOTE_HOST="+r.RemoteAddr)
	}

	if r.TLS != nil {
		newEnv = append(newEnv, "HTTPS=on")
	}

	for k, v := range r.Header {
		k = strings.Map(upperCaseAndUnderscore, k)
		if k == "PROXY" {
			continue
		}
		joinStr := ", "
		if k == "COOKIE" {
			joinStr = "; "
		}
		newEnv = append(newEnv, "HTTP_"+k+"="+strings.Join(v, joinStr))
	}

	if r.ContentLength > 0 {
		newEnv = append(newEnv, fmt.Sprintf("CONTENT_LENGTH=%d", r.ContentLength))
	}
	if ctype := r.Header.Get("Content-Type"); ctype != "" {
		newEnv = append(newEnv, "CONTENT_TYPE="+ctype)
	}

	port := "8080"
	if matches := portRegex.FindStringSubmatch(r.Host); len(matches) != 0 {
		port = matches[1]
	}
	newEnv = append(newEnv, []string{
		"SERVER_NAME=" + r.Host,
		"SERVER_PROTOCOL=HTTP/1.1",
		"HTTP_HOST=" + r.Host,
		"GATEWAY_INTERFACE=CGI/1.1",
		"REQUEST_METHOD=" + r.Method,
		"QUERY_STRING=" + r.URL.RawQuery,
		"REQUEST_URI=" + r.URL.RequestURI(),
		"PATH_INFO=" + r.URL.Path,
		"SERVER_PORT=" + port,
	}...)

	return removeLeadingDuplicates(newEnv)
}

func removeLeadingDuplicates(env []string) (ret []string) {
	for i, e := range env {
		found := false
		if eq := strings.IndexByte(e, '='); eq != -1 {
			keq := e[:eq+1]
			for _, e2 := range env[i+1:] {
				if strings.HasPrefix(e2, keq) {
					found = true
					break
				}
			}
		}
		if !found {
			ret = append(ret, e)
		}
	}
	return
}

var osDefaultInheritEnv = map[string][]string{
	"darwin":  {"DYLD_LIBRARY_PATH"},
	"freebsd": {"LD_LIBRARY_PATH"},
	"hpux":    {"LD_LIBRARY_PATH", "SHLIB_PATH"},
	"irix":    {"LD_LIBRARY_PATH", "LD_LIBRARYN32_PATH", "LD_LIBRARY64_PATH"},
	"linux":   {"LD_LIBRARY_PATH"},
	"openbsd": {"LD_LIBRARY_PATH"},
	"solaris": {"LD_LIBRARY_PATH", "LD_LIBRARY_PATH_32", "LD_LIBRARY_PATH_64"},
	"windows": {"SystemRoot", "COMSPEC", "PATHEXT", "WINDIR"},
}
