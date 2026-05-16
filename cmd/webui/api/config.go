package api

import (
	"net/http"

	"github.com/XingfenD/tag_file/internal/model"
)

// ConfigItem is the JSON representation of a config entry.
type ConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// HandleListConfig responds to GET /api/config.
// Supports ?q=X&page=N for search and pagination.
func (rt *Router) HandleListConfig(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	pg := parsePagination(r, 0, 10)

	var configs []model.Config
	var total int64
	var err error

	offset := (pg.Page - 1) * pg.PageSize
	if q != "" {
		configs, total, err = rt.ConfigSvc.SearchConfigs(q, offset, pg.PageSize)
	} else {
		configs, total, err = rt.ConfigSvc.List(offset, pg.PageSize)
	}
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}

	pg = computePagination(total, pg.Page, pg.PageSize)

	items := make([]ConfigItem, len(configs))
	for i, c := range configs {
		items[i] = ConfigItem{
			Key:   c.ConfigName,
			Value: c.ConfigValue,
		}
	}

	respond(w, "ok", "success", map[string]any{
		"items":      items,
		"pagination": pg,
	})
}

// HandleSetConfig responds to POST /api/config.
// Reads key and value from form data.
func (rt *Router) HandleSetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	if err := rt.ConfigSvc.Set(key, value); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "配置 «"+key+"» 已保存", "success")
}

// HandleUpdateConfig responds to PUT /api/config/{key}.
// Reads value from form data.
func (rt *Router) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value := r.FormValue("value")
	if err := rt.ConfigSvc.Update(key, value); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "配置 «"+key+"» 已更新", "success")
}

// HandleUnsetConfig responds to DELETE /api/config/{key}.
func (rt *Router) HandleUnsetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := rt.ConfigSvc.Delete(key); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "配置 «"+key+"» 已删除", "success")
}
