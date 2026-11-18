package main

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/cmd/webui/handlers"
)

// empty partial 的 dict 方言渲染门：x-show/x-text 表达式经 html/template
// 属性转义后必须仍可被浏览器/Alpine 还原。
func TestEmptyPartial(t *testing.T) {
	render := func(data map[string]any) string {
		tmpl, err := template.New("t").Funcs(handlers.FuncMap).
			ParseFS(templateFS, "templates/partials/empty.html")
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "empty", data); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}

	out := render(map[string]any{
		"Text": "暂无分类", "Icon": "🗃",
		"Hint": "在上方添加第一个分类",
		"Show": "!loading && items.length === 0",
	})
	for _, want := range []string{`x-show="!loading &amp;&amp; items.length === 0"`, "🗃", "empty-hint", "暂无分类"} {
		if !strings.Contains(out, want) {
			t.Errorf("dict render missing %q in %s", want, out)
		}
	}

	out = render(map[string]any{"TextExpr": "error", "Show": "error"})
	if !strings.Contains(out, `x-text="error"`) {
		t.Errorf("TextExpr render missing x-text in %s", out)
	}

	out = render(map[string]any{"Text": "空"})
	if !strings.Contains(out, "&#9888;") {
		t.Errorf("default icon missing in %s", out)
	}
	for _, absent := range []string{"x-show", "empty-hint"} {
		if strings.Contains(out, absent) {
			t.Errorf("plain render should not contain %q: %s", absent, out)
		}
	}
}

// inlineEditCell partial：Alpine 表达式经 html/template 属性转义后，浏览器实体还原
// 必须原样交回 Alpine（Review Focus 4 的渲染侧哨兵）。
func TestInlineEditCellPartial(t *testing.T) {
	tmpl, err := template.New("t").Funcs(handlers.FuncMap).
		ParseFS(templateFS, "templates/partials/inline_edit.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	data := map[string]any{
		"Value":  "item.name",
		"OnSave": "(v) => updateCategory(item, { name: v })",
	}
	if err := tmpl.ExecuteTemplate(&buf, "inlineEditCell", data); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	t.Logf("rendered:\n%s", out)
	for _, want := range []string{
		`inlineEdit({ value: item.name, onSave: (v) =&gt; updateCategory(item, { name: v }) })`,
		`@blur="editing && save()"`,
		`x-ref="input"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}
