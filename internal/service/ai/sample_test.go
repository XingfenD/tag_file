package ai

// thinUniform is unexported and depends on nothing but integers, so its test
// runs in-package instead of through the public surface.

import "testing"

func TestThinUniform(t *testing.T) {
	cases := []struct {
		n, budget int
		want      []int
	}{
		{0, 10, []int{}},
		{3, 10, []int{0, 1, 2}},
		{10, 10, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{8, 3, []int{0, 3, 7}},
		{8, 1, []int{0}},
		{10000, 100, nil}, // 下面单独断言形状
	}
	for _, c := range cases {
		got := thinUniform(c.n, c.budget)
		if c.want != nil {
			if len(got) != len(c.want) {
				t.Fatalf("thinUniform(%d,%d)=%v, want %v", c.n, c.budget, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("thinUniform(%d,%d)=%v, want %v", c.n, c.budget, got, c.want)
				}
			}
			continue
		}
		if len(got) != c.budget || got[0] != 0 || got[len(got)-1] != c.n-1 {
			t.Fatalf("thinUniform(%d,%d) 形状错: len=%d first=%d last=%d", c.n, c.budget, len(got), got[0], got[len(got)-1])
		}
		for i := 1; i < len(got); i++ {
			if got[i] <= got[i-1] {
				t.Fatalf("索引非严格升序: %v", got)
			}
		}
	}
}
