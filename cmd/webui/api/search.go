package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

// SearchResultItem is a single search result: file + its default-group tags.
type SearchResultItem struct {
	File FileItem      `json:"file"`
	Tags []FileTagItem `json:"tags"`
}

// HandleSearch responds to POST /api/search.
// Accepts include, exclude (comma-separated tag names), logic (and/or), page, page_size.
func (rt *Router) HandleSearch(w http.ResponseWriter, r *http.Request) {
	includeStr := r.FormValue("include")
	excludeStr := r.FormValue("exclude")
	logic := r.FormValue("logic")
	if logic == "" {
		logic = "and"
	}

	include := splitTagNames(includeStr)
	exclude := splitTagNames(excludeStr)

	if len(include) == 0 && len(exclude) == 0 {
		respond(w, "请至少输入一个搜索条件", "error")
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

	results, err := rt.SearchSvc.SearchByTags(include, exclude, logic)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}

	total := int64(len(results))
	pg := computePagination(total, page, pageSize)

	// Paginate results
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(results) {
		start = len(results)
	}
	if end > len(results) {
		end = len(results)
	}
	pageResults := results[start:end]

	// Build tag paths: load all tags once
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

// splitTagNames splits a comma-separated tag name string, trimming whitespace.
func splitTagNames(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
