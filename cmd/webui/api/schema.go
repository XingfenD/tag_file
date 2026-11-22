package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// HandleListSchemas responds to GET /api/schemas: every schema with its
// fields, each field annotated with its bound tag's display name.
func (rt *Router) HandleListSchemas(c *gin.Context) {
	schemas, err := rt.SchemaSvc.ListSchemas()
	if err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	// Resolve bind tag names in one pass.
	idSet := make(map[uint]bool)
	for _, s := range schemas {
		for _, f := range s.Fields {
			idSet[f.BindTagID] = true
		}
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	nameOf := map[uint]string{}
	if len(ids) > 0 {
		tags, err := rt.TagSvc.FindByIDs(ids)
		if err != nil {
			respond(c, err.Error(), msgError)
			return
		}
		for _, t := range tags {
			nameOf[t.TagID] = t.TagName
		}
	}

	out := make([]map[string]any, 0, len(schemas))
	for _, s := range schemas {
		fields := make([]map[string]any, 0, len(s.Fields))
		for _, f := range s.Fields {
			name, ok := nameOf[f.BindTagID]
			if !ok {
				name = "（已删除）"
			}
			optCount, _ := rt.SchemaSvc.FieldOptionCount(f.FieldType, f.BindTagID)
			fields = append(fields, map[string]any{
				"field_id": f.FieldID, "label": f.Label, "type": f.FieldType,
				"widget": f.Widget, "bind_tag_id": f.BindTagID, "bind_tag_name": name,
				"sort_order": f.SortOrder, "option_count": optCount,
			})
		}
		out = append(out, map[string]any{
			"schema_id": s.SchemaID, "schema_name": s.SchemaName,
			"description": s.Description, "sort_order": s.SortOrder, "fields": fields,
		})
	}
	respond(c, "ok", "success", map[string]any{"schemas": out})
}

// HandleCreateSchema responds to POST /api/schemas (name, description).
func (rt *Router) HandleCreateSchema(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		respond(c, "场景名不能为空", msgError)
		return
	}
	s, err := rt.SchemaSvc.AddSchema(name, c.PostForm("description"))
	if err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "场景已创建", "success", map[string]any{"schema_id": s.SchemaID})
}

// HandleUpdateSchema responds to PUT /api/schemas/:id (name, description, sort_order).
func (rt *Router) HandleUpdateSchema(c *gin.Context) {
	id, ok := uintParam(c, "id")
	if !ok {
		return
	}
	order, _ := strconv.Atoi(c.PostForm("sort_order"))
	if err := rt.SchemaSvc.UpdateSchema(id, c.PostForm("name"), c.PostForm("description"), order); err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "场景已更新", "success")
}

// HandleDeleteSchema responds to DELETE /api/schemas/:id. Definition only;
// materialized tags and file values survive.
func (rt *Router) HandleDeleteSchema(c *gin.Context) {
	id, ok := uintParam(c, "id")
	if !ok {
		return
	}
	if err := rt.SchemaSvc.RemoveSchema(id); err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "场景已删除", "success")
}

// HandleAddField responds to POST /api/schemas/:id/fields
// (label, type, widget, bind_tag_id).
func (rt *Router) HandleAddField(c *gin.Context) {
	sid, ok := uintParam(c, "id")
	if !ok {
		return
	}
	bind, ok := uintForm(c, "bind_tag_id")
	if !ok {
		return
	}
	f, err := rt.SchemaSvc.AddField(sid, c.PostForm("label"), c.PostForm("type"), c.PostForm("widget"), bind)
	if err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "字段已添加", "success", map[string]any{"field_id": f.FieldID})
}

// HandleUpdateField responds to PUT /api/schemas/fields/:id
// (label, widget, bind_tag_id, sort_order).
func (rt *Router) HandleUpdateField(c *gin.Context) {
	fid, ok := uintParam(c, "id")
	if !ok {
		return
	}
	bind, ok := uintForm(c, "bind_tag_id")
	if !ok {
		return
	}
	order, _ := strconv.Atoi(c.PostForm("sort_order"))
	if err := rt.SchemaSvc.UpdateField(fid, c.PostForm("label"), c.PostForm("widget"), bind, order); err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "字段已更新", "success")
}

// HandleDeleteField responds to DELETE /api/schemas/fields/:id.
func (rt *Router) HandleDeleteField(c *gin.Context) {
	fid, ok := uintParam(c, "id")
	if !ok {
		return
	}
	if err := rt.SchemaSvc.RemoveField(fid); err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "字段已删除", "success")
}

// HandleSchemaPanels responds to GET /api/schemas/panels?file_path=.
func (rt *Router) HandleSchemaPanels(c *gin.Context) {
	path := c.Query("file_path")
	if path == "" {
		respond(c, "文件路径不能为空", msgError)
		return
	}
	panels, err := rt.SchemaSvc.PanelsForFile(path)
	if err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "ok", "success", map[string]any{"panels": panels})
}

// HandleSchemaApply responds to POST /api/schemas/apply
// (file_path, field_id, value; multi uses repeated tag_ids).
func (rt *Router) HandleSchemaApply(c *gin.Context) {
	path := c.PostForm("file_path")
	if path == "" {
		respond(c, "文件路径不能为空", msgError)
		return
	}
	fid, ok := uintForm(c, "field_id")
	if !ok {
		return
	}
	var checked []uint
	for _, raw := range c.PostFormArray("tag_ids") {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || id == 0 {
			// A collapsed/typo'd array must never read as "nothing checked" —
			// for multi that would take over the field and wipe every child row.
			respond(c, "无效的 tag_ids", msgError)
			return
		}
		checked = append(checked, uint(id))
	}
	if err := rt.SchemaSvc.ApplyField(path, fid, c.PostForm("value"), checked); err != nil {
		respond(c, err.Error(), msgError)
		return
	}
	respond(c, "已保存", "success")
}

// uintParam reads a positive uint path parameter, replying 400 on junk.
func uintParam(c *gin.Context, name string) (uint, bool) {
	n, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || n == 0 {
		respond(c, "无效的 "+name, msgError)
		return 0, false
	}
	return uint(n), true
}

// uintForm reads a positive uint form field, replying 400 on junk.
func uintForm(c *gin.Context, name string) (uint, bool) {
	n, err := strconv.ParseUint(c.PostForm(name), 10, 32)
	if err != nil || n == 0 {
		respond(c, "无效的 "+name, msgError)
		return 0, false
	}
	return uint(n), true
}
