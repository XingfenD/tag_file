package api

import (
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/gin-gonic/gin"
)

// ConfigItem is the JSON representation of a config entry.
type ConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// HandleListConfig responds to GET /api/config.
// Supports ?q=X&page=N for search and pagination.
func (rt *Router) HandleListConfig(c *gin.Context) {
	q := c.Query("q")
	pg := parsePagination(c, 0, 10)

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
		respond(c, err.Error(), "error")
		return
	}

	pg = computePagination(total, pg.Page, pg.PageSize)

	items := make([]ConfigItem, len(configs))
	for i, cfg := range configs {
		items[i] = ConfigItem{
			Key:   cfg.ConfigName,
			Value: cfg.ConfigValue,
		}
	}

	respond(c, "ok", "success", map[string]any{
		"items":      items,
		"pagination": pg,
	})
}

// HandleSetConfig responds to POST /api/config.
// Reads key and value from form data.
func (rt *Router) HandleSetConfig(c *gin.Context) {
	key := c.PostForm("key")
	value := c.PostForm("value")
	if err := rt.ConfigSvc.Set(key, value); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "配置 «"+key+"» 已保存", "success")
}

// HandleUpdateConfig responds to PUT /api/config/:key.
// Reads value from form data.
func (rt *Router) HandleUpdateConfig(c *gin.Context) {
	key := c.Param("key")
	value := c.PostForm("value")
	if err := rt.ConfigSvc.Update(key, value); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "配置 «"+key+"» 已更新", "success")
}

// HandleUnsetConfig responds to DELETE /api/config/:key.
func (rt *Router) HandleUnsetConfig(c *gin.Context) {
	key := c.Param("key")
	if err := rt.ConfigSvc.Delete(key); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "配置 «"+key+"» 已删除", "success")
}
