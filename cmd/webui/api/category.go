package api

import (
	"strconv"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/gin-gonic/gin"
)

// CategoryItem is the JSON representation of a tag category.
type CategoryItem struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// HandleListCategories responds to GET /api/categories.
// Supports ?q=<search>&page=N for search and pagination.
func (rt *Router) HandleListCategories(c *gin.Context) {
	q := c.Query("q")
	pg := parsePagination(c, 0, 10)

	items, updatedPg, err := rt.listCategories(q, pg)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	respond(c, "ok", "success", map[string]any{
		"items":      items,
		"pagination": updatedPg,
	})
}

// listCategories is a non-HTTP Go function that returns categorized items with pagination.
// It can be called directly with typed structs (no HTTP concerns).
func (rt *Router) listCategories(q string, pg Pagination) ([]CategoryItem, Pagination, error) {
	var categories []model.TagCategory
	var total int64
	var err error

	offset := (pg.Page - 1) * pg.PageSize
	if q != "" {
		categories, total, err = rt.CategorySvc.SearchCategories(q, offset, pg.PageSize)
	} else {
		categories, total, err = rt.CategorySvc.ListCategories(offset, pg.PageSize)
	}
	if err != nil {
		return nil, pg, err
	}

	updatedPg := computePagination(total, pg.Page, pg.PageSize)

	items := make([]CategoryItem, len(categories))
	for i, c := range categories {
		items[i] = CategoryItem{
			ID:   c.CategoryID,
			Name: c.CategoryName,
			Desc: c.Description,
		}
	}
	return items, updatedPg, nil
}

// HandleAddCategory responds to POST /api/categories.
// Reads name and description from form data.
func (rt *Router) HandleAddCategory(c *gin.Context) {
	name := c.PostForm("name")
	desc := c.PostForm("description")
	cat, err := rt.CategorySvc.AddCategory(name, desc)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "分类 «"+name+"» 已添加", "success", map[string]any{
		"category": map[string]any{
			"id":   cat.CategoryID,
			"name": cat.CategoryName,
			"desc": cat.Description,
		},
	})
}

// HandleUpdateCategory responds to PUT /api/categories/:id.
// Reads name and description from form data.
func (rt *Router) HandleUpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(c, "无效的ID", "error")
		return
	}
	name := c.PostForm("name")
	desc := c.PostForm("description")
	if _, err := rt.CategorySvc.UpdateCategory(uint(id), name, desc); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "分类 «"+name+"» 已更新", "success")
}

// HandleRemoveCategory responds to DELETE /api/categories/:id.
// Looks up the category name by ID, then removes it by name.
func (rt *Router) HandleRemoveCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(c, "无效的ID", "error")
		return
	}

	// List all categories and find the one to delete
	categories, _, _ := rt.CategorySvc.ListCategories(0, 0)
	for _, cat := range categories {
		if cat.CategoryID == uint(id) {
			if err := rt.CategorySvc.RemoveCategory(cat.CategoryName); err != nil {
				respond(c, err.Error(), "error")
				return
			}
			respond(c, "分类 «"+cat.CategoryName+"» 已删除", "success")
			return
		}
	}
	respond(c, "分类不存在", "error")
}
