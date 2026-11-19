package ai

// dir 是未导出字段: 外部包(ai_test)构造不出带临时目录的 ExtractedFrames,
// 所以 Cleanup 的 nil 安全与幂等只能在包内测试。真实 extractor 在成功时把
// 目录移交给调用方, 测试 fake 只填 Paths, 这个方法是唯一的清理入口。

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractedFramesCleanup(t *testing.T) {
	var nilFrames *ExtractedFrames
	nilFrames.Cleanup() // nil 接收者不得 panic

	scratch := filepath.Join(t.TempDir(), "tagfile-ai-test")
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}
	f := &ExtractedFrames{dir: scratch}
	f.Cleanup()
	f.Cleanup() // 幂等: 第二次调用无害
	if _, err := os.Stat(scratch); !os.IsNotExist(err) {
		t.Fatalf("Cleanup 没有删掉 %s", scratch)
	}
}
