package handlers

import (
	"net/http"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Server) HandleListConfig(w http.ResponseWriter, r *http.Request) {
	pg := Paginate(r, 0, 10)
	q := r.URL.Query().Get("q")
	var configs []model.Config
	var total int64
	var err error
	if q != "" {
		configs, total, err = s.ConfigSvc.SearchConfigs(q, pg.Start-1, pg.PageSize)
	} else {
		configs, total, err = s.ConfigSvc.List(pg.Start-1, pg.PageSize)
	}
	if err != nil {
		s.Redirect(w, r, "/config", err.Error(), "error")
		return
	}
	UpdatePagination(&pg, total, len(configs))
	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":    "config",
		"Configs":   configs,
		"PageData":  pg,
		"BaseURL":   PaginationBaseURL(r),
		"Query":     q,
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.Render(w, "config", data)
}

func (s *Server) HandleSetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	if err := s.ConfigSvc.Set(key, value); err != nil {
		s.Redirect(w, r, "/config", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/config", "配置 «"+key+"» 已保存", "success")
}

func (s *Server) HandleUnsetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	if err := s.ConfigSvc.Delete(key); err != nil {
		s.Redirect(w, r, "/config", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/config", "配置 «"+key+"» 已删除", "success")
}

func (s *Server) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	if err := s.ConfigSvc.Update(key, value); err != nil {
		s.Redirect(w, r, "/config", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/config", "配置 «"+key+"» 已更新", "success")
}
