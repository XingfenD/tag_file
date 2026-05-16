package handlers

import "net/http"

func (s *Server) HandleHome(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "home", map[string]any{"Active": "home"})
}
