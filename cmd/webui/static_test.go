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
