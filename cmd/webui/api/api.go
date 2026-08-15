// Package api provides the JSON data layer for the TagFile Web UI.
// It exposes RESTful /api/... endpoints consumed by frontend JavaScript
// and typed Go functions that page handlers can call directly.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/XingfenD/tag_file/internal/model"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	exportSvc "github.com/XingfenD/tag_file/internal/service/export"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

// Pagination holds pagination metadata for list endpoints.
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasPrev    bool  `json:"has_prev"`
	HasNext    bool  `json:"has_next"`
	PrevPage   int   `json:"prev_page"`
	NextPage   int   `json:"next_page"`
	Start      int   `json:"start"`
	End        int   `json:"end"`
}

// Router holds service dependencies and registers API routes.
type Router struct {
	CategorySvc *categorySvc.Service
	TagSvc      *tagSvc.Service
	FileSvc     *fileSvc.Service
	SearchSvc   *searchSvc.Service
	ConfigSvc   *configSvc.Service
	ExportSvc   *exportSvc.Service
}

// NewRouter creates a new API router with the given service dependencies.
func NewRouter(
	cs *categorySvc.Service,
	ts *tagSvc.Service,
	fs *fileSvc.Service,
	ss *searchSvc.Service,
	cfs *configSvc.Service,
	es *exportSvc.Service,
) *Router {
	return &Router{
		CategorySvc: cs,
		TagSvc:      ts,
		FileSvc:     fs,
		SearchSvc:   ss,
		ConfigSvc:   cfs,
		ExportSvc:   es,
	}
}

// RegisterRoutes registers all /api/... routes on the given mux.
// Uses Go 1.22+ routing syntax.
func (rt *Router) RegisterRoutes(mux *http.ServeMux) {
	// Categories
	mux.HandleFunc("GET /api/categories", rt.HandleListCategories)
	mux.HandleFunc("POST /api/categories", rt.HandleAddCategory)
	mux.HandleFunc("PUT /api/categories/{id}", rt.HandleUpdateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", rt.HandleRemoveCategory)

	// Tags
	mux.HandleFunc("GET /api/tags", rt.HandleListTags)
	mux.HandleFunc("POST /api/tags", rt.HandleAddTag)
	mux.HandleFunc("PUT /api/tags/{id}", rt.HandleUpdateTag)
	mux.HandleFunc("DELETE /api/tags/{id}", rt.HandleRemoveTag)
	mux.HandleFunc("GET /api/tags/values/batch", rt.HandleListTagValuesBatch)
	mux.HandleFunc("GET /api/tags/values", rt.HandleListTagValues)
	mux.HandleFunc("POST /api/tags/values", rt.HandleAddTagValue)
	mux.HandleFunc("DELETE /api/tags/values/{id}", rt.HandleRemoveTagValue)

	// Files - tree and management
	mux.HandleFunc("GET /api/files", rt.HandleFileTree)
	mux.HandleFunc("POST /api/files/scan", rt.HandleScanFiles)
	mux.HandleFunc("POST /api/files/refresh", rt.HandleRefreshTree)
	mux.HandleFunc("POST /api/files/import", rt.HandleImportFile)

	// Files - detail and groups (path as URL segment)
	mux.HandleFunc("GET /api/files/detail/{path...}", rt.HandleFileDetail)
	mux.HandleFunc("GET /api/files/groups/{path...}", rt.HandleFileGroups)
	mux.HandleFunc("GET /api/files/raw/{id}", rt.HandleFileRaw)

	// Files - tag operations (file path from form body)
	mux.HandleFunc("POST /api/files/done", rt.HandleFileDone)
	mux.HandleFunc("POST /api/files/tag", rt.HandleTagFile)
	mux.HandleFunc("DELETE /api/files/tag", rt.HandleUntagFile)
	mux.HandleFunc("PUT /api/files/tag", rt.HandleUpdateTagValue)

	// Files - group operations (file path from form body)
	mux.HandleFunc("POST /api/files/group", rt.HandleAddGroup)
	mux.HandleFunc("DELETE /api/files/group/{id}", rt.HandleRemoveGroup)
	mux.HandleFunc("PUT /api/files/group/{id}", rt.HandleRenameGroup)

	// Files - rename (path as URL segment)
	mux.HandleFunc("PUT /api/files/{path...}", rt.HandleRenameFile)

	// Config
	mux.HandleFunc("GET /api/config", rt.HandleListConfig)
	mux.HandleFunc("POST /api/config", rt.HandleSetConfig)
	mux.HandleFunc("PUT /api/config/{key}", rt.HandleUpdateConfig)
	mux.HandleFunc("DELETE /api/config/{key}", rt.HandleUnsetConfig)

	// Export
	mux.HandleFunc("GET /api/export", rt.HandleExport)

	// Search
	mux.HandleFunc("POST /api/search", rt.HandleSearch)
}

const msgError = "error"

// respond writes a JSON response using the standard envelope:
// {"ok":true/false,"msg":"...",...extra}.
// If msgType is "error", sets HTTP status to 400 and ok to false.
func respond(w http.ResponseWriter, msg, msgType string, extra ...map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	isOk := true
	if msgType == msgError {
		w.WriteHeader(http.StatusBadRequest)
		isOk = false
	}
	resp := map[string]any{"ok": isOk, "msg": msg}
	for _, m := range extra {
		for k, v := range m {
			resp[k] = v
		}
	}
	json.NewEncoder(w).Encode(resp)
}

// respondStatus is like respond but with an explicit HTTP status code.
func respondStatus(w http.ResponseWriter, msg, msgType string, status int, extra ...map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	isOk := true
	if msgType == msgError {
		isOk = false
	}
	resp := map[string]any{"ok": isOk, "msg": msg}
	for _, m := range extra {
		for k, v := range m {
			resp[k] = v
		}
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// parsePagination extracts pagination params from an HTTP request.
func parsePagination(r *http.Request, total int64, pageSize int) Pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	return computePagination(total, page, pageSize)
}

// computePagination computes a Pagination struct from raw values.
// Used both by HTTP handlers and direct Go callers.
func computePagination(total int64, page, pageSize int) Pagination {
	if pageSize < 1 {
		pageSize = 10
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	if total > 0 && totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > int(total) {
		end = int(total)
	}

	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		Start:      offset + 1,
		End:        end,
	}
}

// buildTagPath recursively builds the full path string for a tag,
// e.g. "Parent / Child / Tag".
// Depth is limited to 20 to prevent infinite recursion from cycles.
func buildTagPath(tagID uint, idToTag map[uint]*model.Tag, depth int) string {
	if depth > 20 {
		return "..."
	}
	t, ok := idToTag[tagID]
	if !ok {
		return "?"
	}
	if t.ParentID == nil {
		return t.TagName
	}
	parent, ok := idToTag[*t.ParentID]
	if !ok {
		return t.TagName
	}
	return buildTagPath(parent.TagID, idToTag, depth+1) + " / " + t.TagName
}
