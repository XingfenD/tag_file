package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/XingfenD/tag_file/internal/model"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
)

// SearchResultItem is a single search result: file + its default-group tags.
type SearchResultItem struct {
	File FileItem      `json:"file"`
	Tags []FileTagItem `json:"tags"`
}

// HandleSearch responds to POST /api/search.
// Accepts query (JSON string of SearchQuery), page, page_size.
func (rt *Router) HandleSearch(w http.ResponseWriter, r *http.Request) {
	queryStr := r.FormValue("query")
	if queryStr == "" {
		respond(w, "缺少 query 参数", "error")
		return
	}

	var query searchSvc.SearchQuery
	if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
		respond(w, "无效的查询格式: "+err.Error(), "error")
		return
	}

	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.FormValue("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	results, err := rt.SearchSvc.SearchByQuery(query)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}

	total := int64(len(results))
	pg := computePagination(total, page, pageSize)

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(results) {
		start = len(results)
	}
	if end > len(results) {
		end = len(results)
	}
	pageResults := results[start:end]

	allTags, _, _ := rt.TagSvc.ListTags("", 0, 0)
	idToTag := make(map[uint]*model.Tag, len(allTags))
	for i := range allTags {
		idToTag[allTags[i].TagID] = &allTags[i]
	}

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

	respond(w, "ok", "success", map[string]any{
		"results":    items,
		"pagination": pg,
	})
}
