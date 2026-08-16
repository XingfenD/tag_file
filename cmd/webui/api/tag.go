package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
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
func (rt *Router) HandleListTags(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := r.URL.Query().Get("category_id")
	q := r.URL.Query().Get("q")
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "tree"
	}
	if q != "" {
		view = "flat"
	}

	// Resolve category ID to category name
	categories, _, _ := rt.CategorySvc.ListCategories(0, 0)
	categoryName := ""
	if categoryIDStr != "" {
		cid, parseErr := strconv.ParseUint(categoryIDStr, 10, 64)
		if parseErr == nil {
			for _, c := range categories {
				if c.CategoryID == uint(cid) {
					categoryName = c.CategoryName
					break
				}
			}
		}
	}

	// Load all tags for path building
	allTags, _, _ := rt.TagSvc.ListTags("", 0, 0)
	idToTag := make(map[uint]*model.Tag, len(allTags))
	for i := range allTags {
		idToTag[allTags[i].TagID] = &allTags[i]
	}

	if view == "tree" {
		tree, err := rt.TagSvc.ListTagTree(categoryName)
		if err != nil {
			respond(w, err.Error(), "error")
			return
		}
		treeItems := convertTagTree(tree, idToTag)
		respond(w, "ok", "success", map[string]any{
			"view": "tree",
			"tree": treeItems,
		})
		return
	}

	// Flat / paginated view
	pg := parsePagination(r, 0, 10)
	var tags []model.Tag
	var total int64
	var err error

	if q != "" {
		tags, total, err = rt.TagSvc.SearchTags(q, (pg.Page-1)*pg.PageSize, pg.PageSize)
	} else {
		if categoryName != "" {
			tags, total, err = rt.TagSvc.ListTags(categoryName, (pg.Page-1)*pg.PageSize, pg.PageSize)
		} else {
			tags, total, err = rt.TagSvc.ListTags("", (pg.Page-1)*pg.PageSize, pg.PageSize)
		}
	}
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}

	pg = computePagination(total, pg.Page, pg.PageSize)

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

	respond(w, "ok", "success", map[string]any{
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
func (rt *Router) HandleAddTag(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := r.FormValue("category_id")
	tagName := r.FormValue("tag_name")
	desc := r.FormValue("description")
	parentTagName := r.FormValue("parent_tag_name")

	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil {
		respond(w, "无效的分类ID", "error")
		return
	}

	categories, _, _ := rt.CategorySvc.ListCategories(0, 0)
	var categoryName string
	for _, c := range categories {
		if c.CategoryID == uint(categoryID) {
			categoryName = c.CategoryName
			break
		}
	}
	if categoryName == "" {
		respond(w, "分类不存在", "error")
		return
	}

	if _, err := rt.TagSvc.AddTag(categoryName, tagName, desc, parentTagName); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签 «"+tagName+"» 已添加", "success")
}

// HandleUpdateTag responds to PUT /api/tags/{id}.
// Reads tag_name, description, parent_tag_name from form data.
func (rt *Router) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(w, "无效的ID", "error")
		return
	}
	name := r.FormValue("tag_name")
	desc := r.FormValue("description")
	var parentName *string
	if r.Form.Has("parent_tag_name") {
		v := r.FormValue("parent_tag_name")
		parentName = &v
	}
	if _, err := rt.TagSvc.UpdateTag(uint(id), name, desc, parentName); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签 «"+name+"» 已更新", "success")
}

// HandleRemoveTag responds to DELETE /api/tags/{id}.
// Looks up the tag name by ID, then removes it by name.
func (rt *Router) HandleRemoveTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(w, "无效的ID", "error")
		return
	}

	tag, err := rt.TagSvc.FindByID(uint(id))
	if err != nil || tag == nil {
		respond(w, "标签不存在", "error")
		return
	}
	if _, err := rt.TagSvc.RemoveTag(tag.TagName); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签 «"+tag.TagName+"» 已删除", "success")
}

// HandleListTagValues responds to GET /api/tags/values?tag_name=X.
func (rt *Router) HandleListTagValues(w http.ResponseWriter, r *http.Request) {
	tagName := r.URL.Query().Get("tag_name")
	if tagName == "" {
		respond(w, "缺少 tag_name 参数", "error")
		return
	}
	values, err := rt.TagSvc.ListTagValues(tagName)
	if err != nil {
		respond(w, err.Error(), "error")
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

	respond(w, "ok", "success", map[string]any{"values": items})
}

// HandleListTagValuesBatch responds to GET /api/tags/values/batch?names=A,B,C.
func (rt *Router) HandleListTagValuesBatch(w http.ResponseWriter, r *http.Request) {
	namesParam := r.URL.Query().Get("names")
	if namesParam == "" {
		respond(w, "缺少 names 参数", "error")
		return
	}
	names := strings.Split(namesParam, ",")
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}

	result, err := rt.TagSvc.ListTagValuesBatch(names)
	if err != nil {
		respond(w, err.Error(), "error")
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

	respond(w, "ok", "success", map[string]any{"values": valuesMap})
}

// HandleAddTagValue responds to POST /api/tags/values.
// Reads tag_name and value from form data.
func (rt *Router) HandleAddTagValue(w http.ResponseWriter, r *http.Request) {
	tagName := r.FormValue("tag_name")
	val := r.FormValue("value")
	if tagName == "" || val == "" {
		respond(w, "标签名和值不能为空", "error")
		return
	}
	if _, err := rt.TagSvc.AddTagValue(tagName, val); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "值 «"+val+"» 已添加", "success")
}

// HandleRemoveTagValue responds to DELETE /api/tags/values/{id}.
// Reads tag_name and value from form data.
func (rt *Router) HandleRemoveTagValue(w http.ResponseWriter, r *http.Request) {
	tagName := r.FormValue("tag_name")
	val := r.FormValue("value")
	if tagName == "" || val == "" {
		respond(w, "标签名和值不能为空", "error")
		return
	}
	if err := rt.TagSvc.RemoveTagValue(tagName, val); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "值 «"+val+"» 已删除", "success")
}
