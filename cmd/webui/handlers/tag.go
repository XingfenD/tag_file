package handlers

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Server) HandleListTags(w http.ResponseWriter, r *http.Request) {
	categories, _, _ := s.CategorySvc.ListCategories(0, 0)
	categoryIDStr := r.URL.Query().Get("category_id")
	q := r.URL.Query().Get("q")
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "tree"
	}

	// search only works in flat view
	if q != "" {
		view = "flat"
	}

	allTags, _, _ := s.TagSvc.ListTags("", 0, 0)
	idToTag := make(map[uint]*model.Tag, len(allTags))
	for i := range allTags {
		idToTag[allTags[i].TagID] = &allTags[i]
	}

	var buildTagPath func(tagID uint, depth int) string
	buildTagPath = func(tagID uint, depth int) string {
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
		return buildTagPath(parent.TagID, depth+1) + " / " + t.TagName
	}

	allTagsData := make([]map[string]any, 0, len(allTags))
	parentPaths := make(map[uint]string, len(allTags))
	for _, t := range allTags {
		fullPath := buildTagPath(t.TagID, 0)
		allTagsData = append(allTagsData, map[string]any{
			"name": t.TagName,
			"path": fullPath,
			"cat":  t.Category.CategoryName,
			"cid":  t.CategoryID,
			"id":   t.TagID,
		})

		if t.ParentID != nil {
			if parent, ok := idToTag[*t.ParentID]; ok {
				parentPaths[t.TagID] = buildTagPath(parent.TagID, 0)
			}
		}
	}
	sort.Slice(allTagsData, func(i, j int) bool {
		return allTagsData[i]["path"].(string) < allTagsData[j]["path"].(string)
	})

	if view == "tree" {
		categoryName := ""
		if categoryIDStr != "" {
			id, parseErr := ParseInt(categoryIDStr)
			if parseErr == nil {
				for _, c := range categories {
					if c.CategoryID == id {
						categoryName = c.CategoryName
						break
					}
				}
			}
		}
		tree, err := s.TagSvc.ListTagTree(categoryName)
		if err != nil {
			s.Redirect(w, r, "/tags", err.Error(), "error")
			return
		}
		data := map[string]any{
			"Active":     "tags",
			"Categories": categories,
			"TagTree":    tree,
			"AllTags":    allTagsData,
			"View":       "tree",
			"CategoryID": categoryIDStr,
		}
		s.Render(w, "tag", data)
		return
	}

	pg := Paginate(r, 0, 10)
	var tags []model.Tag
	var total int64
	var err error
	if q != "" {
		tags, total, err = s.TagSvc.SearchTags(q, pg.Start-1, pg.PageSize)
	} else {
		if categoryIDStr != "" {
			id, parseErr := ParseInt(categoryIDStr)
			if parseErr == nil {
				for _, c := range categories {
					if c.CategoryID == id {
						tags, total, err = s.TagSvc.ListTags(c.CategoryName, pg.Start-1, pg.PageSize)
						break
					}
				}
			}
		}
		if len(tags) == 0 && categoryIDStr == "" {
			tags, total, err = s.TagSvc.ListTags("", pg.Start-1, pg.PageSize)
		}
	}
	if err != nil {
		s.Redirect(w, r, "/tags", err.Error(), "error")
		return
	}
	UpdatePagination(&pg, total, 0)

	data := map[string]any{
		"Active":      "tags",
		"Categories":  categories,
		"AllTags":     allTagsData,
		"ParentPaths": parentPaths,
		"Tags":        tags,
		"PageData":    pg,
		"BaseURL":     PaginationBaseURL(r),
		"Query":       q,
		"CategoryID":  categoryIDStr,
		"View":        "flat",
	}
	s.Render(w, "tag", data)
}

func (s *Server) HandleAddTag(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := r.FormValue("category_id")
	tagName := r.FormValue("tag_name")
	desc := r.FormValue("description")
	parentTagName := r.FormValue("parent_tag_name")

	categoryID, err := ParseInt(categoryIDStr)
	if err != nil {
		respond(w, "无效的分类ID", "error")
		return
	}

	categories, _, _ := s.CategorySvc.ListCategories(0, 0)
	var categoryName string
	for _, c := range categories {
		if c.CategoryID == categoryID {
			categoryName = c.CategoryName
			break
		}
	}
	if categoryName == "" {
		respond(w, "分类不存在", "error")
		return
	}

	if _, err := s.TagSvc.AddTag(categoryName, tagName, desc, parentTagName); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签 «"+tagName+"» 已添加", "success")
}

func (s *Server) HandleRemoveTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		respond(w, "无效的ID", "error")
		return
	}

	tags, _, _ := s.TagSvc.ListTags("", 0, 0)
	for _, t := range tags {
		if t.TagID == id {
			if _, err := s.TagSvc.RemoveTag(t.TagName); err != nil {
				respond(w, err.Error(), "error")
				return
			}
			respond(w, "标签 «"+t.TagName+"» 已删除", "success")
			return
		}
	}
	respond(w, "标签不存在", "error")
}

func (s *Server) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		respond(w, "无效的ID", "error")
		return
	}
	name := r.FormValue("tag_name")
	desc := r.FormValue("description")
	parentName := r.FormValue("parent_tag_name")
	if _, err := s.TagSvc.UpdateTag(id, name, desc, parentName); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签 «"+name+"» 已更新", "success")
}

func (s *Server) HandleAPITags(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := r.URL.Query().Get("category_id")
	categoryID, err := ParseInt(categoryIDStr)
	if err != nil || categoryID == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}

	categories, _, _ := s.CategorySvc.ListCategories(0, 0)
	var categoryName string
	for _, c := range categories {
		if c.CategoryID == categoryID {
			categoryName = c.CategoryName
			break
		}
	}
	if categoryName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}

	tags, _, _ := s.TagSvc.ListTags(categoryName, 0, 0)

	idToTag := make(map[uint]*model.Tag, len(tags))
	for i := range tags {
		idToTag[tags[i].TagID] = &tags[i]
	}

	var buildPath func(tagID uint, depth int) string
	buildPath = func(tagID uint, depth int) string {
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
		return buildPath(parent.TagID, depth+1) + " / " + t.TagName
	}

	type tagItem struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Path string `json:"path"`
		Cat  string `json:"cat"`
		CID  uint   `json:"cid"`
	}
	items := make([]tagItem, 0, len(tags))
	for i := range tags {
		items = append(items, tagItem{ID: tags[i].TagID, Name: tags[i].TagName, Path: buildPath(tags[i].TagID, 0), Cat: categoryName, CID: tags[i].CategoryID})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Path < items[j].Path
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) HandleTagValuesAPI(w http.ResponseWriter, r *http.Request) {
	tagName := r.URL.Query().Get("tag_name")
	if tagName == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}
	values, err := s.TagSvc.ListTagValues(tagName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}
	type valItem struct {
		ID    uint   `json:"id"`
		Value string `json:"value"`
	}
	items := make([]valItem, 0, len(values))
	for _, v := range values {
		items = append(items, valItem{ID: v.ValueID, Value: v.Value})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) HandleAddTagValue(w http.ResponseWriter, r *http.Request) {
	tagName := r.FormValue("tag_name")
	val := r.FormValue("value")
	if tagName == "" || val == "" {
		respond(w, "标签名和值不能为空", "error")
		return
	}
	if _, err := s.TagSvc.AddTagValue(tagName, val); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "值 «"+val+"» 已添加", "success")
}

func (s *Server) HandleRemoveTagValue(w http.ResponseWriter, r *http.Request) {
	tagName := r.FormValue("tag_name")
	val := r.FormValue("value")
	if tagName == "" || val == "" {
		respond(w, "标签名和值不能为空", "error")
		return
	}
	if err := s.TagSvc.RemoveTagValue(tagName, val); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "值 «"+val+"» 已删除", "success")
}
