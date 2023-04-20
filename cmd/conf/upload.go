package conf

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/google/uuid"
	"github.com/oneshot-uno/oneshot/internal/file"
	"github.com/oneshot-uno/oneshot/internal/handlers"
	"github.com/oneshot-uno/oneshot/internal/server"
)

func (c *Conf) setupUploadRoute(args []string, srvr *server.Server) (*server.Route, error) {
	var filePath string
	if len(args) >= 1 {
		filePath = args[0]
	}

	if c.Dir != "" {
		filePath = filepath.Join(c.Dir, filePath)
	}

	if c.FileName != "" && filePath == "" {
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
	if c.FileName != "" {
		file.SetName(c.FileName, false)
	}

	if !c.NoInfo {
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
	if c.ExitOnFail {
		route.MaxRequests = 1
	} else {
		route.MaxOK = 1
	}

	base := `{{ define "base" }}<!DOCTYPE html>
<html>
<head>
<link rel="apple-touch-icon" href="/assets/icon.png">
<link rel="icon" type="image/png" href="/assets/icon.png">
</head>
<body>
{{ if .FileSection }}
  {{ template "file-section" .CSRFToken }}
{{ end }}
{{ if .InputSection }}
  {{ if .FileSection }}
    <br/>OR<br/>
  {{ end }}
  {{ template "input-section" .CSRFToken }}
{{ end }}
</body>
</html>
{{ end }}
`

	fileSection := `{{ define "file-section" }}<form action="/" method="post" enctype="multipart/form-data">
  {{ if ne . "" }}<input type="hidden" name="csrf-token" value="{{ . }}">{{ end }}
  <h5>Select a file to upload:</h5>
  <input type="file" name="oneshot">
  <br><br>
  <input type="submit" value="Upload">
</form>{{ end }}`

	inputSection := `{{ define "input-section" }}<form action="/" method="post">
  {{ if ne . "" }}<input type="hidden" name="csrf-token" value="{{ . }}">{{ end }}
  <h5>Enter text to send: </h5>
  <textarea name="text"></textarea>
  <br><br>
  <input type="submit" value="Upload">
</form>{{ end }}`

	var (
		err  error
		tmpl *template.Template
	)

	if c.UploadHTML != "" {
		tmpl, err = template.ParseFiles(c.UploadHTML)
		if err != nil {
			return nil, err
		}
	} else {
		tmpl, err = template.New("file-section").Parse(fileSection)
		if err != nil {
			return nil, err
		}
		tmpl, err = tmpl.Parse(inputSection)
		if err != nil {
			return nil, err
		}
		tmpl, err = tmpl.Parse(base)
		if err != nil {
			return nil, err
		}
	}

	sections := struct {
		FileSection  bool
		InputSection bool
		CSRFToken    string
	}{
		FileSection:  true,
		InputSection: true,
		CSRFToken:    c.CustomCSRFToken,
	}

	if sections.CSRFToken == "" && !c.NoCSRFToken {
		sections.CSRFToken = uuid.New().String()
	}

	getHandler := func(w http.ResponseWriter, r *http.Request) error {
		tmpl.ExecuteTemplate(w, "base", &sections)
		return server.OKNotDoneErr
	}
	postHandler := handlers.HandleUpload(file, !c.NoUnixNorm, sections.CSRFToken, srvr.InfoLog)

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
