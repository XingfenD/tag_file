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
	if strings.Contains(html, "dirLeaf") || strings.Contains(html, "dir_path") {
		t.Error("file_detail.html sidebar header must carry only the sort control, no directory breadcrumb")
	}
	wb := read("css/components/workbench.css")
	for _, want := range []string{
		".tree-current-band",
		"border-right: 2px solid var(--accent)",
		"inset 0 0 12px -4px var(--accent-glow)",
		"transition: transform 0.25s ease",
	} {
		if !strings.Contains(wb, want) {
			t.Errorf("workbench.css current-row band missing %q", want)
		}
	}
	if strings.Contains(wb, ".tree-file.tree-current") {
		t.Error("workbench.css must not paint the current row itself — the sliding .tree-current-band is the marker")
	}
	treejs := read("js/core/tree.js")
	for _, want := range []string{"positionBand", "syncBand", "tree-current-band", "fresh.style.animation = 'none'", "offsetTop", "settleBand", "getAnimations"} {
		if !strings.Contains(treejs, want) {
			t.Errorf("core/tree.js missing %q (band slide + no replayed row fade)", want)
		}
	}
	treecss := read("css/components/tree.css")
	for _, want := range []string{"interpolate-size: allow-keywords", "::details-content", "allow-discrete"} {
		if !strings.Contains(treecss, want) {
			t.Errorf("components/tree.css missing %q (directory open/close height animation)", want)
		}
	}
	for _, gone := range []string{"dir-file-list", "dir-indexed", "dir-unindexed"} {
		if strings.Contains(html, gone) {
			t.Errorf("file_detail.html still carries old dir list marker %q", gone)
		}
	}
	js := read("js/pages/file/detail.js")
	if strings.Contains(js, `this.$el.querySelector('#wb-dir-tree')`) {
		t.Error("detail.js treeApi must query #wb-dir-tree from document — onFileClick/onLoaded closures run in the tree scope, where $el is the tree element itself")
	}
	if !strings.Contains(js, `document.getElementById('wb-dir-tree')`) {
		t.Error("detail.js treeApi missing document.getElementById('wb-dir-tree')")
	}
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

// 随机打标入口门：功能入口挂在首页与文件列表工具栏；侧栏导航不含该链接。
func TestRandomTaggingEntry(t *testing.T) {
	b, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "/files/recommend") {
		t.Error("base.html sidebar must not carry the 随机打标 link")
	}
	for _, tpl := range []string{"templates/home.html", "templates/file.html"} {
		tb, err := templateFS.ReadFile(tpl)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(tb), `href="/files/recommend"`) {
			t.Errorf("%s missing the 随机打标 entry", tpl)
		}
	}
}

// 导航折叠门：按钮/label 包裹/根类样式/偏好键四处齐备，且侧栏导航不含该链接。
// 折叠必须是全插值动画：收起态不许用 display:none 或对齐切换（宽度过渡中
// 瞬时重排 = 抽搐），标签走 max-width/opacity。
func TestNavCollapse(t *testing.T) {
	b, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	base := string(b)
	for _, want := range []string{
		`id="nav-collapse"`,
		`class="nav-label"`,
		`class="brand-words"`,
		`class="brand-logo"`,
		`tagfile_sidebar_collapsed`,
		`classList.add('nav-collapsed')`,
	} {
		if !strings.Contains(base, want) {
			t.Errorf("base.html missing %q (collapse button / label wrappers / first-paint restore)", want)
		}
	}
	if strings.Contains(base, `style="width:20px;height:20px`) {
		t.Error("brand logo must be styled by .brand-logo, not inline (inline margin fights the collapsed rail)")
	}
	js, err := staticFS.ReadFile("static/js/core/sidebar.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `classList.toggle('nav-collapsed')`) {
		t.Error("core/sidebar.js must toggle the root nav-collapsed class")
	}
	css, err := staticFS.ReadFile("static/css/layout/layout.css")
	if err != nil {
		t.Fatal(err)
	}
	lcss := string(css)
	if !strings.Contains(lcss, "html.nav-collapsed .sidebar") {
		t.Error("layout.css missing html.nav-collapsed sidebar rules")
	}
	if !strings.Contains(lcss, "html.nav-collapsed .sidebar nav a .nav-label") || !strings.Contains(lcss, "max-width: 0") {
		t.Error("collapsed labels must collapse via max-width/opacity, not display")
	}
	// 版本行连高度一起收（max-height→0），否则折叠栏里留一行空高把页脚撑歪。
	if !strings.Contains(lcss, "html.nav-collapsed .sidebar-version") || !strings.Contains(lcss, "max-height: 0") {
		t.Error("sidebar-version must collapse its height too")
	}
	// 折叠态的恢复入口要看得见：按钮在收起态提高不透明度。
	if !strings.Contains(lcss, "html.nav-collapsed #nav-collapse") {
		t.Error("collapsed rail must keep #nav-collapse discoverable")
	}
	if strings.Contains(lcss, "html.nav-collapsed .brand-text") {
		t.Error("display-toggle collapse rule (.brand-text) must be gone — it causes mid-transition reflow jitter")
	}
}

// 黏性层级门：目录行按 --td 深度吸顶，样式与渲染器两侧齐备。
func TestTreeStickyHierarchy(t *testing.T) {
	js, err := staticFS.ReadFile("static/js/core/tree.js")
	if err != nil {
		t.Fatal(err)
	}
	treesrc := string(js)
	for _, want := range []string{"nodeHtml(n, depth)", `"--td:' + depth`, "nodeHtml(c, depth + 1)", "nodeHtml(n, 0)"} {
		if !strings.Contains(treesrc, want) {
			t.Errorf("core/tree.js missing %q (depth wired into render)", want)
		}
	}
	css, err := staticFS.ReadFile("static/css/components/tree.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"position: sticky", "calc(var(--td, 0) * var(--tree-row-h", ".tree-dir > .tree-summary"} {
		if !strings.Contains(string(css), want) {
			t.Errorf("components/tree.css missing %q (sticky directory rows)", want)
		}
	}
}

// 舞台氛围底门：图片/视频各一层 ambient + 一层 media（计数即断言，
// 音频/fallback 不得加）；主体无圆角是裁决。
func TestStageAmbientBackdrop(t *testing.T) {
	fh, err := templateFS.ReadFile("templates/file_detail.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(fh)
	if n := strings.Count(html, `class="stage-ambient"`); n != 2 {
		t.Errorf("file_detail.html: stage-ambient expected exactly 2 (image+video), got %d", n)
	}
	if n := strings.Count(html, `class="stage-media"`); n != 2 {
		t.Errorf("file_detail.html: stage-media expected exactly 2 (image+video), got %d", n)
	}
	cssB, err := staticFS.ReadFile("static/css/components/workbench.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssB)
	for _, want := range []string{".stage-layer", "display: contents", ".stage-ambient", "object-fit: cover", "inset: -36px", "pointer-events: none", ".stage-media", "object-fit: contain"} {
		if !strings.Contains(css, want) {
			t.Errorf("workbench.css missing %q (ambient stage backdrop)", want)
		}
	}
	start := strings.Index(css, ".stage-media {")
	if start < 0 {
		t.Fatal("workbench.css missing .stage-media rule")
	}
	if body := css[start:strings.Index(css[start:], "}")+start]; strings.Contains(body, "border-radius") {
		t.Error(".stage-media must not carry border-radius (裁决：图片不要圆角)")
	}
}

// 媒体查看器门：vendor 是 PhotoSwipe 5（GLightbox 已撤，不许回流）；
// 图片点击放大、视频走独立展开按钮（不劫持原生控制条）；
// 触控板捏合/滚轮缩放归库内置，页级缩放不再被劫持。
func TestStageLightbox(t *testing.T) {
	for _, f := range []string{"static/vendor/photoswipe-lightbox.esm.min.js", "static/vendor/photoswipe.esm.min.js", "static/vendor/photoswipe.css"} {
		b, err := staticFS.ReadFile(f)
		if err != nil {
			t.Fatalf("vendor missing %s: %v", f, err)
		}
		if len(b) == 0 {
			t.Errorf("vendor %s is empty", f)
		}
	}
	for _, f := range []string{"static/vendor/glightbox.min.js", "static/vendor/glightbox.min.css"} {
		if _, err := staticFS.ReadFile(f); err == nil {
			t.Errorf("vendor %s must be gone (viewer switched to PhotoSwipe)", f)
		}
	}
	b, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "glightbox") {
		t.Error("base.html must not reference glightbox anymore")
	}
	fh, err := templateFS.ReadFile("templates/file_detail.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(fh)
	imgLine := ""
	videoLine := ""
	for _, line := range strings.Split(html, "\n") {
		if strings.Contains(line, `<img class="stage-media"`) {
			imgLine = line
		}
		if strings.Contains(line, `<video controls class="stage-media"`) {
			videoLine = line
		}
	}
	if imgLine == "" || videoLine == "" {
		t.Fatal("file_detail.html: image/video .stage-media lines not found")
	}
	if !strings.Contains(imgLine, `@click="openViewer()"`) {
		t.Error(`image .stage-media must carry @click="openViewer()"`)
	}
	if strings.Contains(videoLine, "@click") {
		t.Error("video .stage-media must not bind @click (native controls own the surface)")
	}
	if !strings.Contains(html, `class="stage-expand"`) {
		t.Fatal("file_detail.html missing stage-expand button")
	}
	expandLine := ""
	for _, line := range strings.Split(html, "\n") {
		if strings.Contains(line, `class="stage-expand"`) {
			expandLine = line
		}
	}
	if !strings.Contains(expandLine, `@click="openViewer()"`) {
		t.Error(`.stage-expand button line must carry @click="openViewer()"`)
	}
	js, err := staticFS.ReadFile("static/js/pages/file/detail.js")
	if err != nil {
		t.Fatal(err)
	}
	djs := string(js)
	for _, want := range []string{"openViewer", "photoswipe-lightbox.esm.min.js", "loadAndOpen", "contentLoad", "type: 'image'", "type: 'video'"} {
		if !strings.Contains(djs, want) {
			t.Errorf("detail.js missing %q", want)
		}
	}
	if strings.Contains(djs, "GLightbox") {
		t.Error("detail.js must not reference GLightbox anymore")
	}
	wb, err := staticFS.ReadFile("static/css/components/workbench.css")
	if err != nil {
		t.Fatal(err)
	}
	wcss := string(wb)
	if !strings.Contains(wcss, "@media (hover: none)") || !strings.Contains(wcss, ".stage-expand:focus-visible") {
		t.Error(".stage-expand must stay reachable on touch (hover:none) and keyboard (:focus-visible)")
	}
	if !strings.Contains(wcss, ".pswp__video") {
		t.Error("workbench.css missing .pswp__video (viewer video mount style)")
	}
}

// 配置目录侧栏门：模板挂 aside+jumpTo、卡片有锚 id、脚本有 IO 与滚动、样式在 config.css。
func TestConfigNavSidebar(t *testing.T) {
	b, err := templateFS.ReadFile("templates/config.html")
	if err != nil {
		t.Fatal(err)
	}
	cfg := string(b)
	for _, want := range []string{"config-nav", `@click.prevent="jumpTo(`, "config-card-export", `class="panel dir-sidebar"`} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config.html missing %q", want)
		}
	}
	cg, err := templateFS.ReadFile("templates/partials/config_group.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cg), `:id="'config-card-' + g.id"`) {
		t.Error("config_group partial root card must carry the config-card anchor id")
	}
	js, err := staticFS.ReadFile("static/js/pages/config/main.js")
	if err != nil {
		t.Fatal(err)
	}
	mjs := string(js)
	for _, want := range []string{"activeNav", "jumpTo", "IntersectionObserver", "scrollIntoView"} {
		if !strings.Contains(mjs, want) {
			t.Errorf("config/main.js missing %q", want)
		}
	}
	css, err := staticFS.ReadFile("static/css/components/config.css")
	if err != nil {
		t.Fatal(err)
	}
	ccss := string(css)
	for _, want := range []string{".config-nav-item", "position: sticky", "scroll-margin-top"} {
		if !strings.Contains(ccss, want) {
			t.Errorf("config.css missing %q", want)
		}
	}
}

// decodePath 与 encodePath 的互逆门：encodePath 按段 encodeURIComponent，decodeURI
// 解不动 %23/%2C 等保留字符，直开 URL 落回的路径与树行 data-file-path 对不上。
func TestDecodePathInvertsEncodePath(t *testing.T) {
	b, err := staticFS.ReadFile("static/js/core/utils.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	if strings.Contains(js, "decodeURI(") {
		t.Error("utils.js decodePath must use decodeURIComponent, not decodeURI")
	}
	if !strings.Contains(js, "decodeURIComponent(") {
		t.Error("utils.js decodePath missing decodeURIComponent")
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
