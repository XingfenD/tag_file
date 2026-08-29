// Package api provides the JSON data layer for the TagFile Web UI.
// It exposes RESTful /api/... endpoints consumed by frontend JavaScript
// and typed Go functions that page handlers can call directly.
package api

import (
	"net/http"
	"strconv"

	"github.com/XingfenD/tag_file/internal/model"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	exportSvc "github.com/XingfenD/tag_file/internal/service/export"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/gin-gonic/gin"
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

// RegisterRoutes registers all /api/... routes on the given gin router group.
func (rt *Router) RegisterRoutes(api *gin.RouterGroup) {
	// Categories
	cats := api.Group("/categories")
	{
		cats.GET("", rt.HandleListCategories)
		cats.POST("", rt.HandleAddCategory)
		cats.PUT("/:id", rt.HandleUpdateCategory)
		cats.DELETE("/:id", rt.HandleRemoveCategory)
	}

	// Tags
	tags := api.Group("/tags")
	{
		tags.GET("", rt.HandleListTags)
		tags.POST("", rt.HandleAddTag)
		tags.PUT("/:id", rt.HandleUpdateTag)
		tags.DELETE("/:id", rt.HandleRemoveTag)
		tags.GET("/values/batch", rt.HandleListTagValuesBatch)
		tags.GET("/values", rt.HandleListTagValues)
		tags.POST("/values", rt.HandleAddTagValue)
		tags.DELETE("/values/:id", rt.HandleRemoveTagValue)
	}

	// Files
	files := api.Group("/files")
	{
		files.GET("", rt.HandleFileTree)
		files.GET("/expand", rt.HandleExpandDir)
		files.POST("/scan", rt.HandleScanFiles)
		files.POST("/refresh", rt.HandleRefreshTree)
		files.POST("/import", rt.HandleImportFile)
		files.GET("/detail/*path", rt.HandleFileDetail)
		files.GET("/groups/*path", rt.HandleFileGroups)
		files.GET("/raw/:id", rt.HandleFileRaw)
		files.POST("/done", rt.HandleFileDone)
		files.POST("/tag", rt.HandleTagFile)
		files.DELETE("/tag", rt.HandleUntagFile)
		files.PUT("/tag", rt.HandleUpdateTagValue)
		files.POST("/group", rt.HandleAddGroup)
		files.DELETE("/group/:id", rt.HandleRemoveGroup)
		files.PUT("/group/:id", rt.HandleRenameGroup)
		files.PUT("/rename", rt.HandleRenameFile)
	}

	// Config
	cfg := api.Group("/config")
	{
		cfg.GET("", rt.HandleListConfig)
		cfg.POST("", rt.HandleSetConfig)
		cfg.PUT("/:key", rt.HandleUpdateConfig)
		cfg.DELETE("/:key", rt.HandleUnsetConfig)
	}

	// Export and Search
	api.GET("/export", rt.HandleExport)
	api.POST("/search", rt.HandleSearch)
}

const msgError = "error"

// respond writes a JSON response using the standard envelope.
func respond(c *gin.Context, msg, msgType string, extra ...map[string]any) {
	isOk := true
	status := http.StatusOK
	if msgType == msgError {
		isOk = false
		status = http.StatusBadRequest
	}
	resp := map[string]any{"ok": isOk, "msg": msg}
	for _, m := range extra {
		for k, v := range m {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// respondStatus is like respond but with an explicit HTTP status code.
func respondStatus(c *gin.Context, msg, msgType string, status int, extra ...map[string]any) {
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
	c.JSON(status, resp)
}

// parsePagination extracts pagination params from a gin context.
func parsePagination(c *gin.Context, total int64, pageSize int) Pagination {
	page, _ := strconv.Atoi(c.Query("page"))
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
