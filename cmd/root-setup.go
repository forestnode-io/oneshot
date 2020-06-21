package cmd

import (
	"fmt"
	ezcgi "github.com/raphaelreyna/ez-cgi/pkg/cgi"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func downloadSetup(cmd *cobra.Command, args []string, srvr *server.Server) *server.Route {
	var filePath string
	if len(args) >= 1 {
		filePath = args[0]
	}
	if filePath != "" && fileName != "" {
		fileName = filepath.Base(filePath)
	}
	file := &server.FileReader{
		Path:     filePath,
		Name:     fileName,
		Ext:      fileExt,
		MimeType: fileMime,
	}

	if !noInfo {
		file.ProgressWriter = os.Stdout
	}

	route := &server.Route{
		Pattern: "/",
		Methods: []string{"GET"},
		DoneHandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("gone"))
		},
	}
	if exitOnFail {
		route.MaxRequests = 1
	} else {
		route.MaxOK = 1
	}
	route.HandlerFunc = handlers.HandleDownload(file, !noDownload, srvr.InfoLog)

	return route
}

func cgiSetup(cmd *cobra.Command, args []string, srvr *server.Server) (*server.Route, error) {
	var err error

	handler := &ezcgi.Handler{
		InheritEnv: envVars,
	}

	argLen := len(args)
	if argLen < 1 {
		if shellCommand {
			return nil, fmt.Errorf("no shell command given\n exit")
		}
		return nil, fmt.Errorf("path to executable not given\n exit")
	}
	if shellCommand {
		handler.Path = shell
		handler.Args = []string{"-c", args[0]}
	} else {
		handler.Path = args[0]
		if argLen >= 2 {
			handler.Args = args[1:argLen]
		}
	}

	if cgiStderr != "" {
		handler.Stderr, err = os.Open(cgiStderr)
		defer handler.Stderr.(io.WriteCloser).Close()
		if err != nil {
			return nil, err
		}
	}

	header := http.Header{}
	for _, rh := range rawHeaders {
		parts := strings.SplitN(rh, ":", 2)
		if len(parts) < 2 {
			err = fmt.Errorf("invalid header: %s", rh)
			return nil, err
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		header.Set(k, v)
	}
	if fileMime != "" {
		header.Set("Content-Type", fileMime)
	}
	var fn string
	if fileName == "" {
		fn = fmt.Sprintf("%0-x", rand.Int31())
		if fileExt != "" {
			fn += strings.ReplaceAll(fileExt, ".", "")
		}
	} else {
		fn = fileName
	}
	if !noDownload {
		header.Set("Content-Disposition",
			fmt.Sprintf("attachment;filename=%s", fn))
	}
	if len(header) != 0 {
		handler.Header = header
	}

	if dir != "" {
		handler.Dir = dir
	} else {
		handler.Dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	if !noError {
		handler.Logger = srvr.ErrorLog
	}

	if replaceHeaders {
		handler.OutputHandler = ezcgi.EZOutputHandlerReplacer
	}

	if cgiStrict {
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
	if exitOnFail {
		route.MaxRequests = 1
	} else {
		route.MaxOK = 1
	}
	route.HandlerFunc = handlers.HandleCGI(handler, fn, fileMime, srvr.InfoLog)

	return route, nil
}
