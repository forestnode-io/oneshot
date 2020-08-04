package conf

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/raphaelreyna/oneshot/internal/file"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/internal/server"
	"io/ioutil"
	"math/rand"
)

func (c *Conf) setupDownloadRoute(args []string, srvr *server.Server) (*server.Route, error) {
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

	if filePath == "" && c.WaitForEOF {
		tdir, err := ioutil.TempDir("", "oneshot")
		if err != nil {
			return nil, err
		}

		if c.FileName == "" {
			c.FileName = fmt.Sprintf("%0-x", rand.Int31())
		}
		filePath = filepath.Join(tdir, c.FileName, c.FileExt)
		c.stdinBufLoc = filePath
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
