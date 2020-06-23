package cmd

import (
	"github.com/raphaelreyna/oneshot/internal/file"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"path/filepath"
)

func downloadSetup(cmd *cobra.Command, args []string, srvr *server.Server) *server.Route {
	var filePath string
	if len(args) >= 1 {
		filePath = args[0]
	}
	if filePath != "" && fileName != "" {
		fileName = filepath.Base(filePath)
	}
	file := &file.FileReader{
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
