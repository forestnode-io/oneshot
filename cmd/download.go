package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/raphaelreyna/oneshot/cmd/conf"
	"github.com/raphaelreyna/oneshot/internal/file"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
)

func downloadSetup(args []string, srvr *server.Server, c *conf.Conf) (*server.Route, error) {
	var filePath string
	if len(args) >= 1 {
		filePath = args[0]
	}
	if filePath != "" && c.FileName == "" {
		c.FileName = filepath.Base(filePath)
	}
	if c.ArchiveMethod != "zip" && c.ArchiveMethod != "tar.gz" {
		c.ArchiveMethod = "tar.gz"
	}
	file := &file.FileReader{
		Path:          filePath,
		Name:          c.FileName,
		Ext:           c.FileExt,
		MimeType:      c.FileMime,
		ArchiveMethod: c.ArchiveMethod,
	}

	if !c.NoInfo {
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
	if c.ExitOnFail {
		route.MaxRequests = 1
	} else {
		route.MaxOK = 1
	}

	header := http.Header{}
	for _, rh := range c.RawHeaders {
		parts := strings.SplitN(rh, ":", 2)
		if len(parts) < 2 {
			err := fmt.Errorf("invalid header: %s", rh)
			return nil, err
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		header.Set(k, v)
	}

	route.HandlerFunc = handlers.HandleDownload(file, !c.NoDownload, header, srvr.InfoLog)

	return route, nil
}
