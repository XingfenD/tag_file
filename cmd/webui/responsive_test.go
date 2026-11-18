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
