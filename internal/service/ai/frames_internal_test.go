// Internal (package ai) tests for the frame extractor's pure helpers.
// They live in the ai package proper because probeDuration, adaptiveInterval,
// extractTimeout and parseCandidateTimes are unexported by design — a
// package ai_test file cannot even name them, let alone pin their numbers.
package ai

import (
	"testing"
	"time"
)

func TestAdaptiveInterval(t *testing.T) {
	cases := []struct {
		config, duration float64
		maxFrames        int
		want             float64
	}{
		{1, 0, 10, 1},     // ffprobe 失败 → 配置间隔
		{1, 6, 5, 1},      // 6/50=0.12 < 1 → 配置间隔
		{0.1, 6, 1, 0.6},  // 6/10=0.6 > 0.1 → 拉伸覆盖全片
		{1, 7200, 10, 72}, // 2 小时电影 / 100 候选 = 72s
	}
	for _, c := range cases {
		if got := adaptiveInterval(c.config, c.duration, c.maxFrames); got != c.want {
			t.Errorf("adaptiveInterval(%v,%v,%d)=%v, want %v", c.config, c.duration, c.maxFrames, got, c.want)
		}
	}
}

func TestExtractTimeout(t *testing.T) {
	cases := []struct {
		duration float64
		want     time.Duration
	}{
		{0, 120 * time.Second},
		{30, 120 * time.Second}, // 60s < floor
		{300, 600 * time.Second},
		{100000, 30 * time.Minute}, // 封顶
	}
	for _, c := range cases {
		if got := extractTimeout(c.duration); got != c.want {
			t.Errorf("extractTimeout(%v)=%v, want %v", c.duration, got, c.want)
		}
	}
}

func TestParseCandidateTimes(t *testing.T) {
	stderr := []byte("[Parsed_showinfo_1 @ 0x1] n:0 pts:0 pts_time:0 fmt:yuv420p\n" +
		"noise line without timestamp\n" +
		"[Parsed_showinfo_1 @ 0x1] n:1 pts:20000 pts_time:2.5 fmt:yuv420p\n" +
		"[Parsed_showinfo_1 @ 0x1] n:2 pts:40000 pts_time:4 fmt:yuv420p\n")
	got := parseCandidateTimes(stderr)
	want := []float64{0, 2.5, 4}
	if len(got) != len(want) {
		t.Fatalf("parseCandidateTimes=%v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("parseCandidateTimes=%v, want %v", got, want)
		}
	}
}
