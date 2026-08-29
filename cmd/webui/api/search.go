package api

import (
	"encoding/json"
	"strconv"

	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	"github.com/gin-gonic/gin"
)

// SearchResultItem is a single search result: file + its default-group tags.
type SearchResultItem struct {
	File FileItem      `json:"file"`
	Tags []FileTagItem `json:"tags"`
}

// HandleSearch responds to POST /api/search.
// Accepts query (JSON string of SearchQuery), page, page_size.
func (rt *Router) HandleSearch(c *gin.Context) {
	queryStr := c.PostForm("query")
	if queryStr == "" {
		respond(c, "缺少 query 参数", "error")
		return
	}

	var query searchSvc.SearchQuery
	if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
		respond(c, "无效的查询格式: "+err.Error(), "error")
		return
	}

	page, _ := strconv.Atoi(c.PostForm("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.PostForm("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	pageResults, total, err := rt.SearchSvc.SearchByQueryPaged(query, offset, pageSize)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	pg := computePagination(total, page, pageSize)

	var allTagIDs []uint
	for _, r := range pageResults {
		for _, t := range r.Tags {
			allTagIDs = append(allTagIDs, t.TagID)
		}
	}
	idToTag := rt.loadTagPathMap(allTagIDs)

	items := make([]SearchResultItem, 0, len(pageResults))
	for _, r := range pageResults {
		fi := FileItem{
			ID:     r.File.FileID,
			Path:   r.File.FilePath,
			Name:   r.File.FileName,
			Size:   r.File.FileSize,
			Type:   r.File.FileType,
			TagCnt: r.File.TagCnt,
		}
		tagItems := make([]FileTagItem, 0, len(r.Tags))
		for _, t := range r.Tags {
			tagItems = append(tagItems, FileTagItem{
				TagID:   t.TagID,
				TagName: t.TagName,
				TagPath: buildTagPath(t.TagID, idToTag, 0),
			})
		}
		items = append(items, SearchResultItem{File: fi, Tags: tagItems})
	}

	respond(c, "ok", "success", map[string]any{
		"results":    items,
		"pagination": pg,
	})
}
