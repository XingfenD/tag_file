package main

import (
	"strings"
	"testing"
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
		{"css/components/tags.css", ".tag-lg"},
		{"js/pages/file/tree.js", "meta.badgeClass"},
	}
	for _, c := range present {
		if !strings.Contains(read(c.file), c.want) {
			t.Errorf("%s missing %q", c.file, c.want)
		}
	}
}
