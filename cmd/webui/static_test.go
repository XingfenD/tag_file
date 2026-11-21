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

// CSS 回归门：钉死暗色主题四修复，防止硬编码浅色值回流。
func TestThemeTokens(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	cases := []struct{ file, absent string }{
		{"css/components/buttons.css", "color: #fff"},
		{"css/components/navigation.css", "color: #fff"},
		{"css/components/buttons.css", "rgba(224, 82, 82"},
		{"css/components/forms.css", "%237e918d"},
		{"css/components/notifications.css", "rgba(34,197,94"},
		{"css/components/notifications.css", "rgba(224,82,82"},
		{"css/components/notifications.css", "rgba(245,158,11"},
	}
	for _, c := range cases {
		if strings.Contains(read(c.file), c.absent) {
			t.Errorf("%s still contains hardcoded %q", c.file, c.absent)
		}
	}
	if !strings.Contains(read("css/core/variables.css"), "--on-accent") {
		t.Error("variables.css missing --on-accent")
	}
	// 裁决记录：mask-image 会掩掉 select 整块绘制，箭头主题化改走 --select-arrow image token。
	for _, tok := range []string{"--select-arrow", "fill='%237e918d'", "fill='%2366837d'"} {
		if !strings.Contains(read("css/core/variables.css"), tok) {
			t.Errorf("variables.css missing %q", tok)
		}
	}
}

// qb-* 组件化回归门：内联样式全部离场，query-builder.css 成为唯一落点。
func TestQueryBuilderCss(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	qb := read("css/components/query-builder.css")
	for _, want := range []string{".qb-group", ".qb-row", ".qb-header", ".qb-val-area"} {
		if !strings.Contains(qb, want) {
			t.Errorf("query-builder.css missing %q", want)
		}
	}
	js := read("js/pages/search/main.js")
	for _, absent := range []string{`style="`, "blur(8px)"} {
		if strings.Contains(js, absent) {
			t.Errorf("search/main.js still contains %q", absent)
		}
	}
	b, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "/static/css/components/query-builder.css") {
		t.Error("base.html does not load query-builder.css")
	}
}

// Wave 16b：输入框宽度档/列宽/form-help 类收编的回归门。
// 页面组装照抄 main.go（base + partials + 页模板），只执行 content 块。
func TestTemplateWidthClasses(t *testing.T) {
	partials := partialTemplates
	base, err := template.New("base").Funcs(handlers.FuncMap).ParseFS(templateFS, "templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	files := append([]string{"templates/category.html"}, partials...)
	tmpl, err := template.Must(base.Clone()).ParseFS(templateFS, files...)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "content", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `style="width:`) {
		t.Error("category content still has inline width styles")
	}
	for _, f := range []struct{ path, want string }{
		{"static/css/components/forms.css", ".input-w-md"},
		{"static/css/components/table.css", ".col-w-100"},
	} {
		b, err := staticFS.ReadFile(f.path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), f.want) {
			t.Errorf("%s missing %s", f.path, f.want)
		}
	}
}

// Wave 16a：标量 token 回归门。裁决记录：plan 的字面 gate（禁一切
// font-size:Npx / blur(Npx)）与其替换映射（7 档字号 + 3 档 blur）矛盾——
// 15/17/36/40px 与 blur 12/16px 是组件身份值，plan 未发 token。
// 故门钉「映射内字面值零残留 + token 齐备 + 光晕配方单源」。
func TestScalarTokens(t *testing.T) {
	vars := ""
	mappedFont := regexp.MustCompile(`font-size:\s*(10|11|12|13|14|16|22)px`)
	mappedBlur := regexp.MustCompile(`blur\((4|8|24)px\)`)
	glow := regexp.MustCompile(`0 (2px 8px|4px 16px|6px 24px) var\(--accent-glow\)`)
	err := fs.WalkDir(staticFS, "static/css", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".css") {
			return err
		}
		b, err := staticFS.ReadFile(path)
		if err != nil {
			return err
		}
		s := string(b)
		if strings.HasSuffix(path, "core/variables.css") {
			vars = s
			return nil
		}
		if strings.Contains(path, "/components/") || strings.HasSuffix(path, "layout/layout.css") {
			for _, re := range []*regexp.Regexp{mappedFont, mappedBlur, glow} {
				if loc := re.FindString(s); loc != "" {
					t.Errorf("%s still has unmapped literal %q", path, loc)
				}
			}
		}
		if strings.Contains(path, "/core/") && strings.HasSuffix(path, "base.css") {
			if loc := mappedFont.FindString(s); loc != "" {
				t.Errorf("core/base.css still has unmapped literal %q", loc)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range []string{
		"--text-2xs: 10px", "--text-xs: 11px", "--text-sm: 12px", "--text-md: 13px",
		"--text-lg: 14px", "--text-xl: 16px", "--text-2xl: 22px",
		"--glass-blur-xs: 4px", "--glass-blur-sm: 8px", "--glass-blur-lg: 24px",
		"--glow-sm: 0 2px 8px var(--accent-glow)", "--glow-md: 0 4px 16px var(--accent-glow)", "--glow-lg: 0 6px 24px var(--accent-glow)",
		"--radius-xs: 4px", "--radius-pill: 999px", "--transition-fast: 0.15s ease",
	} {
		if !strings.Contains(vars, tok) {
			t.Errorf("variables.css missing %q", tok)
		}
	}
}

// Wave 15c：card/panel/toolbar/modal 组件文件落地 + notifications/layout 清空的回归门。
func TestComponentFiles(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	for _, c := range []struct{ file, want string }{
		{"css/components/card.css", ".panel"},
		{"css/components/toolbar.css", ".toolbar"},
		{"css/components/modal.css", ".modal-overlay"},
		{"css/components/forms.css", ".inline-edit-input"},
	} {
		if !strings.Contains(read(c.file), c.want) {
			t.Errorf("%s missing %q", c.file, c.want)
		}
	}
	for _, c := range []struct{ file, gone string }{
		{"css/components/notifications.css", ".modal"},
		{"css/components/notifications.css", ".inline-edit-input"},
		{"css/layout/layout.css", ".card-header"},
		{"css/layout/layout.css", ".card {"},
	} {
		if strings.Contains(read(c.file), c.gone) {
			t.Errorf("%s still contains %q", c.file, c.gone)
		}
	}
	b, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{"card.css", "toolbar.css", "modal.css"} {
		if !strings.Contains(string(b), link) {
			t.Errorf("base.html missing link %s", link)
		}
	}
}

// directoryTree 共享组件的存在性门：树机制的家在 core/tree.js，base.html 必须挂载它。
// （pages/file/tree.js 何时删干净由 Task 2 的单源闸门负责，见 TestDirectoryTreeSingleSource。）
func TestDirectoryTreeComponent(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	js := read("js/core/tree.js")
	for _, want := range []string{
		`Alpine.data('directoryTree'`,
		`tagfile_tree_open:`,
		`data-file-path`,
		`renderToken`,
		`lazyHtml`,
		`cfg.onLoaded`,
		`tree-dir-name`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("core/tree.js missing %q", want)
		}
	}
	// 树数据必须住在工厂闭包的裸数组里：一旦进入 Alpine 响应式图，
	// refreshFileRow 改节点就会触发整树 innerHTML 重建（徽章乐观更新的
	// 「只重绘一行」承诺就此失效，懒追加行也会被冲掉）。
	if strings.Contains(js, "this.tree =") {
		t.Error("core/tree.js assigns tree into reactive state (must stay a raw closure store)")
	}
	fh, err := templateFS.ReadFile("templates/file.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fh), "isEmpty") {
		t.Error("file.html empty-state must read the component's isEmpty")
	}
	b, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "/static/js/core/tree.js") {
		t.Error("base.html missing core/tree.js script link")
	}
}

// 树机制单源闸门：列表页脚本不许再藏渲染/懒加载/重挂逻辑，页面必须挂组件。
// （file_detail.html 的同款断言在 TestWorkbenchSidebarIsDirectoryTree。）
func TestDirectoryTreeSingleSource(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	page := read("js/pages/file/tree.js")
	for _, gone := range []string{"treeHtml", "fileRowHtml", "expandDir"} {
		if strings.Contains(page, gone) {
			t.Errorf("pages/file/tree.js still carries mechanic %q (should live in core/tree.js only)", gone)
		}
	}
	b, err := templateFS.ReadFile("templates/file.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "directoryTree(") {
		t.Error("file.html missing directoryTree( usage")
	}
}

// 工作台侧栏替换门：目录列表标记不得回流，指纹不得再盯未渲染的 dir 字段。
func TestWorkbenchSidebarIsDirectoryTree(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	b, err := templateFS.ReadFile("templates/file_detail.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	if !strings.Contains(html, "directoryTree(") {
		t.Error("file_detail.html sidebar missing directoryTree( usage")
	}
	for _, gone := range []string{"dir-file-list", "dir-indexed", "dir-unindexed"} {
		if strings.Contains(html, gone) {
			t.Errorf("file_detail.html still carries old dir list marker %q", gone)
		}
	}
	js := read("js/pages/file/detail.js")
	start := strings.Index(js, "function detailStamp")
	end := strings.Index(js, "return parts.join")
	if start < 0 || end <= start {
		t.Fatal("detail.js: detailStamp boundary not found")
	}
	body := js[start:end]
	for _, gone := range []string{"dir_files", "dir_unindexed"} {
		if strings.Contains(body, gone) {
			t.Errorf("detailStamp still fingerprints unrendered %q", gone)
		}
	}
}

// 目录侧栏宽度调节门：把手 DOM/样式/脚本三处齐备，宽度走根变量与 localStorage 偏好键。
func TestSidebarResize(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	js := read("js/core/sidebar.js")
	for _, want := range []string{
		`tagfile_dir_width`,
		`--dir-sidebar-w`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("core/sidebar.js missing %q", want)
		}
	}
	css := read("css/layout/layout.css")
	if !strings.Contains(css, "var(--dir-sidebar-w") {
		t.Error("layout.css .dir-sidebar width must read var(--dir-sidebar-w, …)")
	}
	if !strings.Contains(css, ".dir-resize") || !strings.Contains(css, "col-resize") {
		t.Error("layout.css missing .dir-resize handle styles / col-resize cursor")
	}
	b, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "/static/js/core/sidebar.js") {
		t.Error("base.html missing core/sidebar.js script link")
	}
	for _, tpl := range []string{"templates/file_detail.html", "templates/tag.html"} {
		tb, err := templateFS.ReadFile(tpl)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(tb), "dir-resize") {
			t.Errorf("%s missing the dir-resize handle", tpl)
		}
	}
}

// 死 CSS 清除与同义变体合并的回归门：删掉的东西不许回流。
func TestDeadCssPurge(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	absent := []struct{ file, gone string }{
		{"css/layout/layout.css", ".scan-result"},
		{"css/layout/layout.css", "@keyframes fadeUpIn"},
		{"css/layout/layout.css", ".dir-sidebar-static"},
		{"css/components/config.css", "config-invalid-badge"},
		{"css/components/tree.css", "tree-status-ai-"},
		{"css/components/tags.css", ".tag-chip"},
		{"css/components/tags.css", ".tag-lg"},
		{"css/components/tags.css", ".tag-preview"},
		{"css/components/card.css", ".media-preview"},
		{"js/core/utils.js", "treeClass"},
	}
	for _, c := range absent {
		if strings.Contains(read(c.file), c.gone) {
			t.Errorf("%s still contains %q", c.file, c.gone)
		}
	}
	present := []struct{ file, want string }{
		{"css/core/animations.css", "@keyframes fadeUpIn"},
		{"css/layout/layout.css", ".dir-sidebar.scroll"},
		{"js/core/tree.js", "meta.badgeClass"},
	}
	for _, c := range present {
		if !strings.Contains(read(c.file), c.want) {
			t.Errorf("%s missing %q", c.file, c.want)
		}
	}
}
