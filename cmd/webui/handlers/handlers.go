package handlers

import (
	"html/template"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

var FuncMap = template.FuncMap{
	"formatSize":  formatSize,
	"iconForType": iconForType,
	"urlEncode":   url.PathEscape,
}

func iconForType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "video/"):
		return "🎬"
	case strings.HasPrefix(mimeType, "image/"):
		return "🖼"
	case strings.HasPrefix(mimeType, "audio/"):
		return "🎵"
	case strings.Contains(mimeType, "pdf"):
		return "📄"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "tar") || strings.Contains(mimeType, "rar") || strings.Contains(mimeType, "gzip"):
		return "📦"
	default:
		return "📄"
	}
}

type Server struct {
	pages        map[string]*template.Template
	CoreVersion  string
	WebuiVersion string
}

func NewServer(pages map[string]*template.Template) *Server {
	return &Server{
		pages: pages,
	}
}

func (s *Server) RegisterRoutes(r *gin.Engine) {
	r.GET("/", s.HandleHome)
	r.GET("/categories", s.HandlePage("categories", "category"))
	r.GET("/tags", s.HandlePage("tags", "tag"))
	r.GET("/files", s.HandlePage("files", "file"))
	r.GET("/files/detail/*path", s.HandlePage("files", "file_detail"))
	r.GET("/files/groups/*path", s.HandlePage("files", "file_groups"))
	r.GET("/search", s.HandlePage("search", "search"))
	r.GET("/config", s.HandlePage("config", "config"))
}

func (s *Server) HandlePage(active, page string) gin.HandlerFunc {
	return func(c *gin.Context) {
		s.Render(c, page, map[string]any{"Active": active})
	}
}

func (s *Server) Render(c *gin.Context, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["CoreVersion"] = s.CoreVersion
	data["WebuiVersion"] = s.WebuiVersion
	tmpl := s.pages[page]
	if tmpl == nil {
		c.String(500, "template not found: "+page)
		return
	}
	c.Render(200, NewTemplateRenderer(s.pages).Instance(page, data))
}
