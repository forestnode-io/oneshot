package server

import (
	"net/http"
	"path/filepath"
	"mime"
	"io"
	"fmt"
	"os"
	"time"
	"context"
)

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	if s.done {
		w.Write([]byte("resource is no longer available"))
		return
	}
	s.done = true
	s.mutex.Unlock()

	file, err := os.Open(s.FilePath)
	defer file.Close()
	if err != nil {
		if s.ErrorLog != nil {
			s.ErrorLog.Println(err.Error())
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileName := filepath.Base(s.FilePath)
	fileExt := filepath.Ext(s.FilePath)
	contentType := mime.TypeByExtension(fileExt)

	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment;filename=%s", fileName),
	)
	w.Header().Set("Content-Type", contentType)

	before := time.Now()
	_, err = io.Copy(w, file)
	duration := time.Since(before)
	if s.ErrorLog != nil && err != nil {
		s.ErrorLog.Println(err.Error())
		return
	}

	if s.InfoLog != nil && err == nil {
		s.InfoLog.Printf("%s was downloaded at %s in %s by %s\n",
			s.FilePath,
			before.String(),
			duration.String(),
			r.RemoteAddr,
		)
	}

	s.timer.Stop()
	go s.server.Shutdown(context.Background())
	s.Done <- struct{}{}
}
