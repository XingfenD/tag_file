package file

import (
	"strings"
	"testing"
)

// 媒体扩展必须解析成 MIME 型（video/… audio/…），不能退化为字面扩展名——
// WebUI 的媒体分类与可播映射都只认 MIME 前缀。断言前缀而非精确值：环境自带
// MIME 表时 TypeByExtension 先命中，给出的合法别名（如 video/x-quicktime）同样可用。
func TestFileTypeByExtResolvesMediaTypes(t *testing.T) {
	cases := []struct{ path, wantPrefix string }{
		{"dir/clip.MP4", "video/"},
		{"clip.mp4", "video/"},
		{"clip.m4v", "video/"},
		{"clip.mov", "video/"},
		{"clip.webm", "video/"},
		{"song.mp3", "audio/"},
		{"song.m4a", "audio/"},
		{"voice.wav", "audio/"},
	}
	for _, c := range cases {
		if got := fileTypeByExt(c.path); !strings.HasPrefix(got, c.wantPrefix) {
			t.Errorf("fileTypeByExt(%q) = %q, want prefix %q", c.path, got, c.wantPrefix)
		}
	}
}

// 表外类型维持既有约定：识别不了就存小写扩展名（.zzqq 选在任何环境表里都没有的名字）。
func TestFileTypeByExtUnknownFallsBackToExtension(t *testing.T) {
	if got := fileTypeByExt("report.ZZQQ"); got != ".zzqq" {
		t.Errorf("fileTypeByExt(unknown ext) = %q, want %q", got, ".zzqq")
	}
}
