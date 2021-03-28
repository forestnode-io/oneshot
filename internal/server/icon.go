package server

import (
	_ "embed"
	"net/http"
)

//go:embed icon.png
var icon []byte

func (s *Server) HandleIcon(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Clone().Set("Content-Type", "image/png")
	w.Write(icon)
}
