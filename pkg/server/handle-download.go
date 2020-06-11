package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	if s.done {
		w.Write([]byte("resource is no longer available"))
		return
	}
	s.done = true
	s.mutex.Unlock()

	err := s.file.Open()
	defer s.file.Close()
	if err != nil {
		if s.ErrorLog != nil {
			s.ErrorLog.Println(err.Error())
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment;filename=%s", s.file.Name),
	)
	w.Header().Set("Content-Type", s.file.MimeType)

	before := time.Now()
	_, err = io.Copy(w, s.file)
	duration := time.Since(before)
	if s.ErrorLog != nil && err != nil {
		s.ErrorLog.Println(err.Error())
		return
	}

	if s.InfoLog != nil && err == nil {
		s.InfoLog.Printf("%s was downloaded at %s in %s by %s\n",
			s.file.Name,
			before.String(),
			duration.String(),
			r.RemoteAddr,
		)
	}

	// Stop() method needs to run on seperate goroutine.
	// Otherwise, we deadlock since http.Server.Shutdown()
	// wont return until this function returns.
	go s.Stop(context.Background())
}
