package cmd

import (
	"github.com/raphaelreyna/oneshot/internal/file"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

func uploadSetup(cmd *cobra.Command, args []string, srvr *server.Server) (*server.Route, error) {
	var filePath string
	if len(args) >= 1 {
		filePath = args[0]
	}

	if dir != "" {
		filePath = filepath.Join(dir, filePath)
	}

	if fileName != "" && filePath == "" {
		filePath = "."
	}

	// Does user have permission to write to path?
	if filePath != "" {
		tempFilePath := filepath.Join(filePath, "oneshot_permissions_check")
		tf, err := os.Create(tempFilePath)
		defer func() {
			tf.Close()
			os.Remove(tempFilePath)
		}()
		if err != nil {
			return nil, err
		}
	}

	file := &file.FileWriter{
		Path: filePath,
	}
	if fileName != "" {
		file.SetName(fileName, false)
	}

	if !noInfo {
		file.ProgressWriter = os.Stdout
	}

	if file.Path == "" {
		file.ProgressWriter = nil
		srvr.InfoLog = nil
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

	html := []byte(`<!DOCTYPE html>
<html>
<body>

<form action="/" method="post" enctype="multipart/form-data">
  Select file to upload:
  <input type="file" name="oneshot">
  <input type="submit" value="Upload">
</form>

</body>
</html>`)
	getHandler := func(w http.ResponseWriter, r *http.Request) error {
		w.Write(html)
		return server.OKNotDoneErr
	}
	postHandler := handlers.HandleUpload(file, srvr.InfoLog)

	infoLog := func(format string, v ...interface{}) {
		if srvr.InfoLog != nil {
			srvr.InfoLog.Printf(format, v...)
		}
	}

	dontLog := map[string]struct{}{}
	dlm := &sync.Mutex{}
	route.HandlerFunc = func(w http.ResponseWriter, r *http.Request) error {
		switch r.Method {
		case "POST":
			dlm.Lock()
			_, skip := dontLog[r.RemoteAddr]
			dlm.Unlock()
			if !skip {
				infoLog("connected: %s", r.RemoteAddr)
			}
			return postHandler(w, r)
		case "GET":
			infoLog("connected: %s", r.RemoteAddr)
			err := getHandler(w, r)
			if err == server.OKNotDoneErr {
				dlm.Lock()
				dontLog[r.RemoteAddr] = struct{}{}
				dlm.Unlock()
			}
			return err
		}
		return nil
	}

	return route, nil
}
