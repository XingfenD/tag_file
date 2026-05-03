package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	attrSvc "github.com/XingfenD/tag_file/internal/service/attr"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

var FuncMap = template.FuncMap{
	"formatSize": formatSize,
}

type Server struct {
	pages       map[string]*template.Template
	CategorySvc *categorySvc.Service
	TagSvc      *tagSvc.Service
	AttrSvc     *attrSvc.Service
	FileSvc     *fileSvc.Service
	SearchSvc   *searchSvc.Service
	ConfigSvc   *configSvc.Service
}

func NewServer(pages map[string]*template.Template,
	categorySvc *categorySvc.Service,
	tagSvc *tagSvc.Service,
	attrSvc *attrSvc.Service,
	fileSvc *fileSvc.Service,
	searchSvc *searchSvc.Service,
	configSvc *configSvc.Service,
) *Server {
	return &Server{
		pages:       pages,
		CategorySvc: categorySvc,
		TagSvc:      tagSvc,
		AttrSvc:     attrSvc,
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
	mux.HandleFunc("GET /tags", s.HandleListTags)
	mux.HandleFunc("POST /tags/add", s.HandleAddTag)
	mux.HandleFunc("POST /tags/remove", s.HandleRemoveTag)
	mux.HandleFunc("GET /attributes", s.HandleListAttributes)
	mux.HandleFunc("POST /attributes/add", s.HandleAddAttribute)
	mux.HandleFunc("POST /attributes/remove", s.HandleRemoveAttribute)
	mux.HandleFunc("POST /attributes/values/add", s.HandleAddValue)
	mux.HandleFunc("POST /attributes/values/remove", s.HandleRemoveValue)
	mux.HandleFunc("GET /files", s.HandleListFiles)
	mux.HandleFunc("POST /files/scan", s.HandleScanFiles)
	mux.HandleFunc("POST /files/import", s.HandleImportFile)
	mux.HandleFunc("POST /files/tag", s.HandleTagFile)
	mux.HandleFunc("POST /files/untag", s.HandleUntagFile)
	mux.HandleFunc("GET /files/info", s.HandleFileInfo)
	mux.HandleFunc("GET /files/detail/{id}", s.HandleFileDetail)
	mux.HandleFunc("GET /files/raw", s.HandleFileRaw)
	mux.HandleFunc("GET /search", s.HandleSearchForm)
	mux.HandleFunc("POST /search", s.HandleSearch)
	mux.HandleFunc("GET /config", s.HandleListConfig)
	mux.HandleFunc("POST /config/set", s.HandleSetConfig)
	mux.HandleFunc("POST /config/unset", s.HandleUnsetConfig)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) Redirect(w http.ResponseWriter, r *http.Request, urlPath, msg, msgType string) {
	if msg != "" {
		sep := "?"
		if strings.Contains(urlPath, "?") {
			sep = "&"
		}
		urlPath = fmt.Sprintf("%s%sflash=%s&flash_type=%s", urlPath, sep, url.QueryEscape(msg), url.QueryEscape(msgType))
	}
	http.Redirect(w, r, urlPath, http.StatusSeeOther)
}

func GetFlash(r *http.Request) (string, string) {
	return r.URL.Query().Get("flash"), r.URL.Query().Get("flash_type")
}
