package handlers

import "net/http"

func (s *Server) HandleListTags(w http.ResponseWriter, r *http.Request) {
	categories, _, _ := s.CategorySvc.ListCategories(0, 0)
	categoryIDStr := r.URL.Query().Get("category_id")

	pg := Paginate(r, 0, 10)
	var tags any
	var total int64
	var err error
	var filterCategory uint
	if categoryIDStr != "" {
		id, parseErr := ParseInt(categoryIDStr)
		if parseErr == nil {
			filterCategory = id
			for _, c := range categories {
				if c.CategoryID == id {
					tags, total, err = s.TagSvc.ListTags(c.CategoryName, pg.Start-1, pg.PageSize)
					break
				}
			}
		}
	}
	if tags == nil {
		tags, total, err = s.TagSvc.ListTags("", pg.Start-1, pg.PageSize)
	}
	if err != nil {
		s.Redirect(w, r, "/tags", err.Error(), "error")
		return
	}
	UpdatePagination(&pg, total, 0)

	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":         "tags",
		"Categories":     categories,
		"Tags":           tags,
		"FilterCategory": filterCategory,
		"PageData":       pg,
		"BaseURL":        PaginationBaseURL(r),
		"Flash":          flash,
		"FlashType":      flashType,
	}
	s.Render(w, "tag", data)
}

func (s *Server) HandleAddTag(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := r.FormValue("category_id")
	tagName := r.FormValue("tag_name")
	desc := r.FormValue("description")

	categoryID, err := ParseInt(categoryIDStr)
	if err != nil {
		s.Redirect(w, r, "/tags", "无效的分类ID", "error")
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
		s.Redirect(w, r, "/tags", "分类不存在", "error")
		return
	}

	if _, err := s.TagSvc.AddTag(categoryName, tagName, desc); err != nil {
		s.Redirect(w, r, "/tags", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/tags", "标签 «"+tagName+"» 已添加", "success")
}

func (s *Server) HandleRemoveTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/tags", "无效的ID", "error")
		return
	}

	tags, _, _ := s.TagSvc.ListTags("", 0, 0)
	for _, t := range tags {
		if t.TagID == id {
			if _, err := s.TagSvc.RemoveTag(t.TagName); err != nil {
				s.Redirect(w, r, "/tags", err.Error(), "error")
				return
			}
			s.Redirect(w, r, "/tags", "标签 «"+t.TagName+"» 已删除", "success")
			return
		}
	}
	s.Redirect(w, r, "/tags", "标签不存在", "error")
}
