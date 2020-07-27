package conf

import (
	"fmt"
	ezcgi "github.com/raphaelreyna/ez-cgi/pkg/cgi"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

func (c *Conf) cgiSetup(args []string, srvr *server.Server) (*server.Route, error) {
	var err error

	handler := &ezcgi.Handler{
		InheritEnv: c.EnvVars,
	}

	argLen := len(args)
	if argLen < 1 {
		if c.ShellCommand {
			return nil, fmt.Errorf("no shell command given\n exit")
		}
		return nil, fmt.Errorf("path to executable not given\n exit")
	}
	if c.ShellCommand {
		handler.Path = c.Shell
		handler.Args = []string{"-c", args[0]}
	} else {
		handler.Path = args[0]
		if argLen >= 2 {
			handler.Args = args[1:argLen]
		}
	}

	if c.CgiStderr != "" {
		handler.Stderr, err = os.Open(c.CgiStderr)
		defer handler.Stderr.(io.WriteCloser).Close()
		if err != nil {
			return nil, err
		}
	}

	header := http.Header{}
	for _, rh := range c.RawHeaders {
		parts := strings.SplitN(rh, ":", 2)
		if len(parts) < 2 {
			err = fmt.Errorf("invalid header: %s", rh)
			return nil, err
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		header.Set(k, v)
	}
	if c.FileMime != "" {
		header.Set("Content-Type", c.FileMime)
	}
	var fn string
	if c.FileName == "" {
		fn = fmt.Sprintf("%0-x", rand.Int31())
		if c.FileExt != "" {
			fn += strings.ReplaceAll(c.FileExt, ".", "")
		}
	} else {
		fn = c.FileName
	}
	if !c.NoDownload {
		header.Set("Content-Disposition",
			fmt.Sprintf("attachment;filename=%s", fn))
	}
	if len(header) != 0 {
		handler.Header = header
	}

	if c.Dir != "" {
		handler.Dir = c.Dir
	} else {
		handler.Dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	if !c.NoError {
		handler.Logger = srvr.ErrorLog
	}

	if c.ReplaceHeaders {
		handler.OutputHandler = ezcgi.EZOutputHandlerReplacer
	}

	if c.CgiStrict {
		handler.OutputHandler = ezcgi.DefaultOutputHandler
	}

	if handler.OutputHandler == nil {
		handler.OutputHandler = ezcgi.EZOutputHandler
	}

	route := &server.Route{
		Pattern: "/",
		Methods: []string{"GET", "POST"},
		DoneHandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("gone"))
		},
	}
	if c.ExitOnFail {
		route.MaxRequests = 1
	} else {
		route.MaxOK = 1
	}
	route.HandlerFunc = handlers.HandleCGI(handler, fn, c.FileMime, srvr.InfoLog)

	return route, nil
}
