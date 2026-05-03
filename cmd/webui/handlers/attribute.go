package handlers

import (
	"net/http"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Server) HandleListAttributes(w http.ResponseWriter, r *http.Request) {
	tags, _, _ := s.TagSvc.ListTags("", 0, 0)
	tagIDStr := r.URL.Query().Get("tag_id")
	attrValuesForStr := r.URL.Query().Get("attr_values_for")
	q := r.URL.Query().Get("q")

	pg := Paginate(r, 0, 10)
	var attributes []model.Attribute
	var total int64
	if q != "" {
		var tagName string
		for _, t := range tags {
			if t.TagID == uint(0) {
				break
			}
		}
		if tagIDStr != "" {
			id, _ := ParseInt(tagIDStr)
			for _, t := range tags {
				if t.TagID == id {
					tagName = t.TagName
					break
				}
			}
		}
		attributes, total, _ = s.AttrSvc.SearchAttributes(q, tagName, pg.Start-1, pg.PageSize)
	} else if tagIDStr != "" {
		id, parseErr := ParseInt(tagIDStr)
		if parseErr == nil {
			for _, t := range tags {
				if t.TagID == id {
					attributes, total, _ = s.AttrSvc.ListAttributes(t.TagName, pg.Start-1, pg.PageSize)
					break
				}
			}
		}
	}
	if len(attributes) == 0 && q == "" {
		attributes, total, _ = s.AttrSvc.ListAttributes("", pg.Start-1, pg.PageSize)
	}
	UpdatePagination(&pg, total, 0)

	var attrValues any
	if attrValuesForStr != "" {
		attrID, err := ParseInt(attrValuesForStr)
		if err == nil {
			attrValues, _, _ = s.AttrSvc.ListValues(attrID, 0, 0)
		}
	}

	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":     "attributes",
		"Tags":       tags,
		"Attributes": attributes,
		"AttrValues": attrValues,
		"PageData":   pg,
		"BaseURL":    PaginationBaseURL(r),
		"Query":      q,
		"Flash":      flash,
		"FlashType":  flashType,
	}
	s.Render(w, "attribute", data)
}

func (s *Server) HandleAddAttribute(w http.ResponseWriter, r *http.Request) {
	tagIDStr := r.FormValue("tag_id")
	attrName := r.FormValue("attr_name")
	desc := r.FormValue("description")

	tagID, err := ParseInt(tagIDStr)
	if err != nil {
		s.Redirect(w, r, "/attributes", "无效的标签ID", "error")
		return
	}

	tags, _, _ := s.TagSvc.ListTags("", 0, 0)
	var tagName string
	for _, t := range tags {
		if t.TagID == tagID {
			tagName = t.TagName
			break
		}
	}
	if tagName == "" {
		s.Redirect(w, r, "/attributes", "标签不存在", "error")
		return
	}

	if _, err := s.AttrSvc.AddAttribute(tagName, attrName, desc); err != nil {
		s.Redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/attributes", "属性 «"+attrName+"» 已添加", "success")
}

func (s *Server) HandleRemoveAttribute(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/attributes", "无效的ID", "error")
		return
	}
	if err := s.AttrSvc.RemoveAttribute(id); err != nil {
		s.Redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/attributes", "属性已删除", "success")
}

func (s *Server) HandleUpdateAttribute(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/attributes", "无效的ID", "error")
		return
	}
	name := r.FormValue("attr_name")
	desc := r.FormValue("description")
	if _, err := s.AttrSvc.UpdateAttribute(id, name, desc); err != nil {
		s.Redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/attributes", "属性 «"+name+"» 已更新", "success")
}

func (s *Server) HandleAddValue(w http.ResponseWriter, r *http.Request) {
	attrIDStr := r.FormValue("attr_id")
	valueName := r.FormValue("value_name")

	attrID, err := ParseInt(attrIDStr)
	if err != nil {
		s.Redirect(w, r, "/attributes", "无效的属性ID", "error")
		return
	}
	if _, err := s.AttrSvc.AddValue(attrID, valueName); err != nil {
		s.Redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/attributes", "属性值 «"+valueName+"» 已添加", "success")
}

func (s *Server) HandleRemoveValue(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/attributes", "无效的ID", "error")
		return
	}
	if err := s.AttrSvc.RemoveValue(id); err != nil {
		s.Redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/attributes", "属性值已删除", "success")
}

func (s *Server) HandleUpdateValue(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/attributes", "无效的ID", "error")
		return
	}
	name := r.FormValue("value_name")
	if _, err := s.AttrSvc.UpdateValue(id, name); err != nil {
		s.Redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/attributes", "属性值 «"+name+"» 已更新", "success")
}
