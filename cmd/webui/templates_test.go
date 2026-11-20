package main

import (
	"bytes"
	"html/template"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/cmd/webui/handlers"
)

// empty partial 的 dict 方言渲染门：x-show/x-text 表达式经 html/template
// 属性转义后必须仍可被浏览器/Alpine 还原。
func TestEmptyPartial(t *testing.T) {
	render := func(data map[string]any) string {
		tmpl, err := template.New("t").Funcs(handlers.FuncMap).
			ParseFS(templateFS, "templates/partials/icon.html", "templates/partials/empty.html")
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
		"Text": "暂无分类", "Icon": "layers",
		"Hint": "在上方添加第一个分类",
		"Show": "!loading && items.length === 0",
	})
	for _, want := range []string{`x-show="!loading &amp;&amp; items.length === 0"`, `href="#i-layers"`, "empty-hint", "暂无分类"} {
		if !strings.Contains(out, want) {
			t.Errorf("dict render missing %q in %s", want, out)
		}
	}

	out = render(map[string]any{"TextExpr": "error", "Show": "error"})
	if !strings.Contains(out, `x-text="error"`) {
		t.Errorf("TextExpr render missing x-text in %s", out)
	}

	out = render(map[string]any{"Text": "空"})
	if !strings.Contains(out, `href="#i-warning"`) {
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

// 图标系统的一致性闸门：sprite 里定义的每个 #i-<name> 都要有人调用，每个调用点
// （模板的 dict "Name"、JS 的 iconHtml('…')）也都要在 sprite 里存在。断链的 <use>
// 在浏览器里只是安静地画不出东西，所以只能由测试钉住。
func TestIconSpriteMatchesEveryUsage(t *testing.T) {
	sprite, err := templateFS.ReadFile("templates/partials/icons.html")
	if err != nil {
		t.Fatal(err)
	}
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`id="i-([a-z0-9-]+)"`).FindAllStringSubmatch(string(sprite), -1) {
		defined[m[1]] = true
	}
	if len(defined) == 0 {
		t.Fatal("sprite 里一个图标都没有")
	}

	var bodies []string
	walk := func(fsys fs.FS, root, suffix string) {
		if err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, suffix) {
				return nil
			}
			b, readErr := fs.ReadFile(fsys, p)
			if readErr != nil {
				return readErr
			}
			bodies = append(bodies, string(b))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	walk(templateFS, "templates", ".html")
	walk(staticFS, "static/js", ".js")
	all := strings.Join(bodies, "\n")

	calls := regexp.MustCompile(`dict "Name" "([a-z0-9-]+)"|iconHtml\('([a-z0-9-]+)'`)
	used := map[string]bool{}
	for _, m := range calls.FindAllStringSubmatch(all, -1) {
		name := m[1]
		if name == "" {
			name = m[2]
		}
		used[name] = true
		if !defined[name] {
			t.Errorf("调用了 sprite 里不存在的图标 i-%s", name)
		}
	}

	// 死图标检查放宽一档：iconForMime/iconForName 只返回裸名，不算上面的调用点，
	// 但只要以字符串出现过就认为有人用。
	for name := range defined {
		if used[name] || strings.Contains(all, `"`+name+`"`) || strings.Contains(all, `'`+name+`'`) {
			continue
		}
		t.Errorf("sprite 里的 i-%s 没有任何调用点", name)
	}
}
