package main

import (
	"regexp"
	"strings"
	"testing"
)

// 响应式卡片堆叠不变量：所有会被 ≤768px 堆叠 CSS 命中的 <td>（以及为保持
// 整文件规则而顺带标注的键值表 td）必须携带 data-label——窄屏卡片内的
// 字段名由 table.css 的 td::before { content: attr(data-label) } 渲染，
// 漏一格手机上就少一行标签。清单与 data-label 值来源见
// docs/superpowers/plans/2026-11-18-webui-responsive-layout.md Task 2。
func TestStackedTableDataLabels(t *testing.T) {
	tdOpen := regexp.MustCompile(`(?s)<td[^>]*>`)

	files := []struct {
		name string
		read func() (string, error)
	}{
		{"templates/category.html", func() (string, error) {
			b, err := templateFS.ReadFile("templates/category.html")
			return string(b), err
		}},
		{"templates/tag.html", func() (string, error) {
			b, err := templateFS.ReadFile("templates/tag.html")
			return string(b), err
		}},
		{"templates/file_groups.html", func() (string, error) {
			b, err := templateFS.ReadFile("templates/file_groups.html")
			return string(b), err
		}},
		{"templates/file_detail.html", func() (string, error) {
			b, err := templateFS.ReadFile("templates/file_detail.html")
			return string(b), err
		}},
		{"static/js/pages/review/main.js", func() (string, error) {
			b, err := staticFS.ReadFile("static/js/pages/review/main.js")
			return string(b), err
		}},
	}

	for _, f := range files {
		src, err := f.read()
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		for _, td := range tdOpen.FindAllString(src, -1) {
			if !strings.Contains(td, "data-label") {
				t.Errorf("%s: <td> missing data-label: %.80q", f.name, td)
			}
		}
	}
}

// 评审哨兵 1：堆叠 CSS 作用域是 .table-wrap table——计划点名的四张表若丢了
// 包裹容器（file_groups 曾如此），窄屏上它们仍是挤压的桌面表，data-label 全空转。
func TestStackedTablesWrappedInTableWrap(t *testing.T) {
	for _, name := range []string{
		"templates/category.html",
		"templates/tag.html",
		"templates/file_groups.html",
		"templates/file_detail.html",
	} {
		b, err := templateFS.ReadFile(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(string(b), `class="table-wrap"`) {
			t.Errorf("%s: stacked table not inside a .table-wrap container", name)
		}
	}
}

// 评审哨兵 2：.tag 胶囊是 white-space:nowrap，堆叠卡内长标签名/预设值会把卡片
// 撑出屏幕（Review Focus 2 的幸存者）。堆叠上下文里必须给它让出换行。
// 键值表预览胶囊（file_detail）同理，住 tags.css。
func TestStackedTagChipsCanWrap(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if !strings.Contains(read("css/components/table.css"), "tbody td .tag") {
		t.Error(`table.css stacked media missing "tbody td .tag" wrap allowance`)
	}
	if !strings.Contains(read("css/components/tags.css"), ".tag-preview .tag") {
		t.Error(`tags.css missing ".tag-preview .tag" narrow-screen wrap allowance`)
	}
}

// 评审哨兵 3：行 hover 高亮只有一个来源（(hover:hover) 内）。基础态残留会让
// 触屏首触模拟 hover 留下粘住的染色行。
func TestRowHoverHighlightSingleSource(t *testing.T) {
	b, err := staticFS.ReadFile("static/css/components/table.css")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), "tbody tr:hover td"); n != 1 {
		t.Errorf("table.css has %d sources of \"tbody tr:hover td\", want 1 (hover-gated only)", n)
	}
}
