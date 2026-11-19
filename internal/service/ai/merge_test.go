package ai

import "testing"

func TestMergeRawTags(t *testing.T) {
	cases := []struct {
		name string
		in   []RawTag
		want []RawTag
	}{
		{"空输入", nil, []RawTag{}},
		{"跨帧精确重复折叠", []RawTag{{"主体", "猫", "橘色"}, {"主体", "猫", "橘色"}}, []RawTag{{"主体", "猫", "橘色"}}},
		{"同标签不同值都保留", []RawTag{{"颜色", "裤子", "黑色"}, {"颜色", "裤子", "白色"}}, []RawTag{{"颜色", "裤子", "黑色"}, {"颜色", "裤子", "白色"}}},
		{"空值让位带值_空在前", []RawTag{{"主体", "猫", ""}, {"主体", "猫", "橘色"}}, []RawTag{{"主体", "猫", "橘色"}}},
		{"空值让位带值_空在后", []RawTag{{"主体", "猫", "橘色"}, {"主体", "猫", ""}}, []RawTag{{"主体", "猫", "橘色"}}},
		{"无带值兄弟时空值保留", []RawTag{{"主体", "猫", ""}}, []RawTag{{"主体", "猫", ""}}},
		{"保序_首次出现", []RawTag{{"主体", "猫", "橘色"}, {"材质", "皮革", ""}, {"主体", "猫", "橘色"}}, []RawTag{{"主体", "猫", "橘色"}, {"材质", "皮革", ""}}},
	}
	for _, c := range cases {
		got := mergeRawTags(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
			}
		}
	}
}
