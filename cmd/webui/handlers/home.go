package handlers

import "github.com/gin-gonic/gin"

func (s *Server) HandleHome(c *gin.Context) {
	s.Render(c, "home", map[string]any{"Active": "home"})
}
