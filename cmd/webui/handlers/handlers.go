package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// Server is a page-only rendering shell. Data operations are handled
// by the api package (/api/... routes).
type Server struct {
	pages        map[string]*template.Template
	CoreVersion  string
	WebuiVersion string
}

// NewServer creates a Server with the given page templates.
func NewServer(pages map[string]*template.Template) *Server {
	return &Server{
		pages: pages,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", s.HandleHome)
	mux.HandleFunc("GET /categories", s.HandlePage("categories", "category"))
	mux.HandleFunc("GET /tags", s.HandlePage("tags", "tag"))
	mux.HandleFunc("GET /files", s.HandlePage("files", "file"))
	mux.HandleFunc("GET /files/detail/{path...}", s.HandlePage("files", "file_detail"))
	mux.HandleFunc("GET /files/groups/{path...}", s.HandlePage("files", "file_groups"))
	mux.HandleFunc("GET /search", s.HandlePage("search", "search"))
	mux.HandleFunc("GET /config", s.HandlePage("config", "config"))
}

// HandlePage returns a handler that renders an empty page shell.
func (s *Server) HandlePage(active, page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.Render(w, page, map[string]any{"Active": active})
	}
}

func (s *Server) Render(w http.ResponseWriter, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["CoreVersion"] = s.CoreVersion
	data["WebuiVersion"] = s.WebuiVersion
	tmpl := s.pages[page]
	if tmpl == nil {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		fmt.Fprintf(os.Stderr, "Render error: %v\n", err)
	}
}
