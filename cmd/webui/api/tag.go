package api

import (
	"sort"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/gin-gonic/gin"
)

// TagItem is the JSON representation of a tag.
type TagItem struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	CID  uint   `json:"cid"`
}

// TagTreeItem is a recursive JSON representation for tag tree views.
type TagTreeItem struct {
	Tag      TagItem        `json:"tag"`
	Children []*TagTreeItem `json:"children,omitempty"`
}

// HandleListTags responds to GET /api/tags.
// Supports ?category_id=X&q=X&view=tree|flat&page=N.
// If view=tree, returns tree JSON. If q is set, forces flat view with search.
// If flat, returns paginated list with full tag paths.
func (rt *Router) HandleListTags(c *gin.Context) {
	categoryIDStr := c.Query("category_id")
	q := c.Query("q")
	view := c.Query("view")
	if view == "" {
		view = "tree"
	}
	if q != "" {
		view = "flat"
	}

	categoryName := ""
	if categoryIDStr != "" {
		cid, parseErr := strconv.ParseUint(categoryIDStr, 10, 64)
		if parseErr == nil {
			cat, err := rt.CategorySvc.FindByID(uint(cid))
			if err == nil && cat != nil {
				categoryName = cat.CategoryName
			}
		}
	}

	if view == "tree" {
		allTags, _, _ := rt.TagSvc.ListTags("", 0, 0)
		idToTag := make(map[uint]*model.Tag, len(allTags))
		for i := range allTags {
			idToTag[allTags[i].TagID] = &allTags[i]
		}
		tree, err := rt.TagSvc.ListTagTree(categoryName)
		if err != nil {
			respond(c, err.Error(), "error")
			return
		}
		treeItems := convertTagTree(tree, idToTag)
		respond(c, "ok", "success", map[string]any{
			"view": "tree",
			"tree": treeItems,
		})
		return
	}

	pg := parsePagination(c, 0, 10)
	var tags []model.Tag
	var total int64
	var err error

	if q != "" {
		tags, total, err = rt.TagSvc.SearchTags(q, (pg.Page-1)*pg.PageSize, pg.PageSize)
	} else if categoryName != "" {
		tags, total, err = rt.TagSvc.ListTags(categoryName, (pg.Page-1)*pg.PageSize, pg.PageSize)
	} else {
		tags, total, err = rt.TagSvc.ListTags("", (pg.Page-1)*pg.PageSize, pg.PageSize)
	}
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	pg = computePagination(total, pg.Page, pg.PageSize)

	tagIDs := make([]uint, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.TagID
	}
	idToTag := rt.loadTagPathMap(tagIDs)

	tagItems := make([]TagItem, len(tags))
	for i, t := range tags {
		tagItems[i] = TagItem{
			ID:   t.TagID,
			Name: t.TagName,
			Path: buildTagPath(t.TagID, idToTag, 0),
			CID:  t.CategoryID,
		}
	}
	sort.Slice(tagItems, func(i, j int) bool {
		return tagItems[i].Path < tagItems[j].Path
	})

	respond(c, "ok", "success", map[string]any{
		"view":       "flat",
		"items":      tagItems,
		"pagination": pg,
	})
}

// convertTagTree recursively converts service-level TagTreeNode to API-level TagTreeItem.
func convertTagTree(nodes []tagSvc.TagTreeNode, idToTag map[uint]*model.Tag) []*TagTreeItem {
	items := make([]*TagTreeItem, 0, len(nodes))
	for _, n := range nodes {
		path := buildTagPath(n.Tag.TagID, idToTag, 0)
		item := &TagTreeItem{
			Tag: TagItem{
				ID:   n.Tag.TagID,
				Name: n.Tag.TagName,
				Path: path,
				CID:  n.Tag.CategoryID,
			},
		}
		if len(n.Children) > 0 {
			item.Children = convertTagTree(n.Children, idToTag)
		}
		items = append(items, item)
	}
	return items
}

// HandleAddTag responds to POST /api/tags.
// Reads category_id, tag_name, description, parent_tag_name from form data.
func (rt *Router) HandleAddTag(c *gin.Context) {
	categoryIDStr := c.PostForm("category_id")
	tagName := c.PostForm("tag_name")
	desc := c.PostForm("description")
	parentTagName := c.PostForm("parent_tag_name")

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil {
		respond(c, "无效的分类ID", "error")
		return
	}

	cat, err := rt.CategorySvc.FindByID(uint(categoryID))
	if err != nil || cat == nil {
		respond(c, "分类不存在", "error")
		return
	}
	categoryName := cat.CategoryName

	if _, err := rt.TagSvc.AddTag(categoryName, tagName, desc, parentTagName); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签 «"+tagName+"» 已添加", "success")
}

// HandleUpdateTag responds to PUT /api/tags/:id.
// Reads tag_name, description, parent_tag_name from form data.
func (rt *Router) HandleUpdateTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(c, "无效的ID", "error")
		return
	}
	name := c.PostForm("tag_name")
	desc := c.PostForm("description")
	var parentName *string
	if c.PostForm("parent_tag_name") != "" {
		v := c.PostForm("parent_tag_name")
		parentName = &v
	}
	if _, err := rt.TagSvc.UpdateTag(uint(id), name, desc, parentName); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签 «"+name+"» 已更新", "success")
}

// HandleRemoveTag responds to DELETE /api/tags/:id.
// Looks up the tag name by ID, then removes it by name.
func (rt *Router) HandleRemoveTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(c, "无效的ID", "error")
		return
	}

	tag, err := rt.TagSvc.FindByID(uint(id))
	if err != nil || tag == nil {
		respond(c, "标签不存在", "error")
		return
	}
	if err := rt.TagSvc.RemoveTagByID(uint(id)); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签 «"+tag.TagName+"» 已删除", "success")
}

// HandleListTagValues responds to GET /api/tags/values?tag_name=X.
func (rt *Router) HandleListTagValues(c *gin.Context) {
	tagName := c.Query("tag_name")
	if tagName == "" {
		respond(c, "缺少 tag_name 参数", "error")
		return
	}
	values, err := rt.TagSvc.ListTagValues(tagName)
	if err != nil {
		respond(c, err.Error(), "error")
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

	respond(c, "ok", "success", map[string]any{"values": items})
}

// HandleListTagValuesBatch responds to GET /api/tags/values/batch?names=A,B,C.
func (rt *Router) HandleListTagValuesBatch(c *gin.Context) {
	namesParam := c.Query("names")
	if namesParam == "" {
		respond(c, "缺少 names 参数", "error")
		return
	}
	names := strings.Split(namesParam, ",")
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}

	result, err := rt.TagSvc.ListTagValuesBatch(names)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	type valItem struct {
		ID    uint   `json:"id"`
		Value string `json:"value"`
	}
	valuesMap := make(map[string][]valItem, len(result))
	for tagName, vals := range result {
		items := make([]valItem, 0, len(vals))
		for _, v := range vals {
			items = append(items, valItem{ID: v.ValueID, Value: v.Value})
		}
		valuesMap[tagName] = items
	}

	respond(c, "ok", "success", map[string]any{"values": valuesMap})
}

// HandleAddTagValue responds to POST /api/tags/values.
// Reads tag_name and value from form data.
func (rt *Router) HandleAddTagValue(c *gin.Context) {
	tagName := c.PostForm("tag_name")
	val := c.PostForm("value")
	if tagName == "" || val == "" {
		respond(c, "标签名和值不能为空", "error")
		return
	}
	if _, err := rt.TagSvc.AddTagValue(tagName, val); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "值 «"+val+"» 已添加", "success")
}

// HandleRemoveTagValue responds to DELETE /api/tags/values/:id.
// Reads tag_name and value from form data.
func (rt *Router) HandleRemoveTagValue(c *gin.Context) {
	tagName := c.PostForm("tag_name")
	val := c.PostForm("value")
	if tagName == "" || val == "" {
		respond(c, "标签名和值不能为空", "error")
		return
	}
	if err := rt.TagSvc.RemoveTagValue(tagName, val); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "值 «"+val+"» 已删除", "success")
}
