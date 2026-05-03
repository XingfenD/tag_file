package handlers

import (
	"net/http"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Server) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	pg := Paginate(r, 0, 10)
	q := r.URL.Query().Get("q")
	var categories []model.TagCategory
	var total int64
	var err error
	if q != "" {
		categories, total, err = s.CategorySvc.SearchCategories(q, pg.Start-1, pg.PageSize)
	} else {
		categories, total, err = s.CategorySvc.ListCategories(pg.Start-1, pg.PageSize)
	}
	if err != nil {
		s.Redirect(w, r, "/categories", err.Error(), "error")
		return
	}
	UpdatePagination(&pg, total, len(categories))
	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":     "categories",
		"Categories": categories,
		"PageData":   pg,
		"BaseURL":    PaginationBaseURL(r),
		"Query":      q,
		"Flash":      flash,
		"FlashType":  flashType,
	}
	s.Render(w, "category", data)
}

func (s *Server) HandleAddCategory(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	desc := r.FormValue("description")
	if _, err := s.CategorySvc.AddCategory(name, desc); err != nil {
		s.Redirect(w, r, "/categories", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/categories", "分类 «"+name+"» 已添加", "success")
}

func (s *Server) HandleRemoveCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/categories", "无效的ID", "error")
		return
	}
	categories, _, _ := s.CategorySvc.ListCategories(0, 0)
	for _, c := range categories {
		if c.CategoryID == id {
			if err := s.CategorySvc.RemoveCategory(c.CategoryName); err != nil {
				s.Redirect(w, r, "/categories", err.Error(), "error")
				return
			}
			s.Redirect(w, r, "/categories", "分类 «"+c.CategoryName+"» 已删除", "success")
			return
		}
	}
	s.Redirect(w, r, "/categories", "分类不存在", "error")
}

func (s *Server) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/categories", "无效的ID", "error")
		return
	}
	name := r.FormValue("name")
	desc := r.FormValue("description")
	if _, err := s.CategorySvc.UpdateCategory(id, name, desc); err != nil {
		s.Redirect(w, r, "/categories", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/categories", "分类 «"+name+"» 已更新", "success")
}
