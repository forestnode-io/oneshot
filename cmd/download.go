package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/raphaelreyna/oneshot/internal/file"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
)

func downloadSetup(cmd *cobra.Command, args []string, srvr *server.Server) (*server.Route, error) {
	var filePath string
	if len(args) >= 1 {
		filePath = args[0]
	}
	if filePath != "" && fileName == "" {
		fileName = filepath.Base(filePath)
	}
	if archiveMethod != "zip" && archiveMethod != "tar.gz" {
		archiveMethod = "tar.gz"
	}
	file := &file.FileReader{
		Path:          filePath,
		Name:          fileName,
		Ext:           fileExt,
		MimeType:      fileMime,
		ArchiveMethod: archiveMethod,
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

	header := http.Header{}
	for _, rh := range rawHeaders {
		parts := strings.SplitN(rh, ":", 2)
		if len(parts) < 2 {
			err := fmt.Errorf("invalid header: %s", rh)
			return nil, err
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		header.Set(k, v)
	}

	route.HandlerFunc = handlers.HandleDownload(file, !noDownload, header, srvr.InfoLog)

	return route, nil
}
