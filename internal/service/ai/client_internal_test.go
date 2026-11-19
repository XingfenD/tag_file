package ai

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// bodySnippet 从 ollama.go 移入 client.go 时唯一未被现有测试钉住的行为：
// 中文错误页按 rune 截断，不能切在字节中间。
func TestBodySnippetCutsOnRuneBoundary(t *testing.T) {
	got := bodySnippet([]byte(strings.Repeat("模", visionBodySnippet+50)))
	runes := []rune(got)
	if len(runes) != visionBodySnippet+1 {
		t.Fatalf("snippet = %d runes, want %d (200 + 省略号)", len(runes), visionBodySnippet+1)
	}
	if !utf8.ValidString(got) {
		t.Fatal("snippet 不是合法 UTF-8: 切在了字节中间")
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("snippet 应以 … 结尾, got %q", got)
	}
	if got := bodySnippet([]byte("  hi  ")); got != "hi" {
		t.Errorf("bodySnippet 短 body = %q, want trimmed %q", got, "hi")
	}
}
