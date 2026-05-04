package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"

	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
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
	pages       map[string]*template.Template
	CategorySvc *categorySvc.Service
	TagSvc      *tagSvc.Service
	FileSvc     *fileSvc.Service
	SearchSvc   *searchSvc.Service
	ConfigSvc   *configSvc.Service
}

func NewServer(pages map[string]*template.Template,
	categorySvc *categorySvc.Service,
	tagSvc *tagSvc.Service,
	fileSvc *fileSvc.Service,
	searchSvc *searchSvc.Service,
	configSvc *configSvc.Service,
) *Server {
	return &Server{
		pages:       pages,
		CategorySvc: categorySvc,
		TagSvc:      tagSvc,
		FileSvc:     fileSvc,
		SearchSvc:   searchSvc,
		ConfigSvc:   configSvc,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", s.HandleHome)
	mux.HandleFunc("GET /categories", s.HandleListCategories)
	mux.HandleFunc("POST /categories/add", s.HandleAddCategory)
	mux.HandleFunc("POST /categories/remove", s.HandleRemoveCategory)
	mux.HandleFunc("POST /categories/update", s.HandleUpdateCategory)
	mux.HandleFunc("GET /tags", s.HandleListTags)
	mux.HandleFunc("POST /tags/add", s.HandleAddTag)
	mux.HandleFunc("POST /tags/remove", s.HandleRemoveTag)
	mux.HandleFunc("POST /tags/update", s.HandleUpdateTag)
	mux.HandleFunc("GET /api/tags", s.HandleAPITags)
	mux.HandleFunc("GET /api/tags/values", s.HandleTagValuesAPI)
	mux.HandleFunc("POST /tags/values/add", s.HandleAddTagValue)
	mux.HandleFunc("POST /tags/values/remove", s.HandleRemoveTagValue)
	mux.HandleFunc("GET /files", s.HandleListFiles)
	mux.HandleFunc("POST /files/scan", s.HandleScanFiles)
	mux.HandleFunc("POST /files/refresh", s.HandleRefreshTree)
	mux.HandleFunc("POST /files/import", s.HandleImportFile)
	mux.HandleFunc("POST /files/tag", s.HandleTagFile)
	mux.HandleFunc("POST /files/untag", s.HandleUntagFile)
	mux.HandleFunc("POST /files/rename", s.HandleRenameFile)
	mux.HandleFunc("GET /files/info", s.HandleFileInfo)
	mux.HandleFunc("GET /files/detail/{path...}", s.HandleFileDetail)
	mux.HandleFunc("GET /files/raw", s.HandleFileRaw)
	mux.HandleFunc("GET /search", s.HandleSearchForm)
	mux.HandleFunc("POST /search", s.HandleSearch)
	mux.HandleFunc("GET /config", s.HandleListConfig)
	mux.HandleFunc("POST /config/set", s.HandleSetConfig)
	mux.HandleFunc("POST /config/unset", s.HandleUnsetConfig)
	mux.HandleFunc("POST /config/update", s.HandleUpdateConfig)
}

func (s *Server) Render(w http.ResponseWriter, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Flash"]; !ok {
		data["Flash"] = ""
	}
	if _, ok := data["FlashType"]; !ok {
		data["FlashType"] = ""
	}
	tmpl := s.pages[page]
	if tmpl == nil {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		fmt.Fprintf(os.Stderr, "Render error: %v\n", err)
	}
}

func (s *Server) Redirect(w http.ResponseWriter, r *http.Request, urlPath, msg, msgType string) {
	ret := r.FormValue("_return")
	if ret != "" {
		ret = cleanQuery(ret)
		if ret != "" {
			sep := "?"
			if strings.Contains(urlPath, "?") {
				sep = "&"
			}
			urlPath += sep + ret
		}
	}
	if msg != "" {
		sep := "?"
		if strings.Contains(urlPath, "?") {
			sep = "&"
		}
		urlPath = fmt.Sprintf("%s%sflash=%s&flash_type=%s", urlPath, sep, url.QueryEscape(msg), url.QueryEscape(msgType))
	}
	http.Redirect(w, r, urlPath, http.StatusSeeOther)
}

func cleanQuery(raw string) string {
	var parts []string
	for _, p := range strings.Split(raw, "&") {
		if strings.HasPrefix(p, "flash=") || strings.HasPrefix(p, "flash_type=") {
			continue
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, "&")
}

func GetFlash(r *http.Request) (string, string) {
	return r.URL.Query().Get("flash"), r.URL.Query().Get("flash_type")
}
