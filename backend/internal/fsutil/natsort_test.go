package fsutil

import (
	"slices"
	"testing"
)

func TestCompareNatural(t *testing.T) {
	// less: 期望 a < b 的用例；equal: 期望比较结果为 0
	less := []struct {
		a, b string
	}{
		// 数字段按数值比较：1, 2, ..., 9, 10, 11
		{"1.mp3", "2.mp3"},
		{"2.mp3", "10.mp3"},
		{"9.mp3", "10.mp3"},
		{"10.mp3", "11.mp3"},
		{"19.mp3", "20.mp3"},
		{"99.mp3", "100.mp3"},
		// 中文名内嵌数字
		{"1.【试听】冰雪奇缘·01.mp3", "2.【试听】冰雪奇缘·02·安娜得救了.mp3"},
		{"2.【试听】冰雪奇缘·02·安娜得救了.mp3", "10.冰雪奇缘·10·向北山出发.mp3"},
		{"第9集", "第10集"},
		// 前缀关系
		{"a", "a1"},
		{"song1", "song1a"},
		// 多段数字
		{"a1b2", "a1b10"},
		{"cd1/01.mp3", "cd2/01.mp3"},
		{"cd2/09.mp3", "cd10/01.mp3"},
		// 数值相等时位数少者在前
		{"1", "01"},
		{"01", "001"},
		{"track1.mp3", "track01.mp3"},
		// 空字符串最小
		{"", "0"},
		{"", "a"},
	}

	for _, tt := range less {
		if got := CompareNatural(tt.a, tt.b); got >= 0 {
			t.Errorf("CompareNatural(%q, %q) = %d, 期望 < 0", tt.a, tt.b, got)
		}
		// 同时验证反方向
		if got := CompareNatural(tt.b, tt.a); got <= 0 {
			t.Errorf("CompareNatural(%q, %q) = %d, 期望 > 0（反对称）", tt.b, tt.a, got)
		}
	}

	equal := [][2]string{
		{"abc", "abc"},
		{"", ""},
	}
	for _, tt := range equal {
		if got := CompareNatural(tt[0], tt[1]); got != 0 {
			t.Errorf("CompareNatural(%q, %q) = %d, 期望 0", tt[0], tt[1], got)
		}
	}
}

func TestCompareNaturalSortOrder(t *testing.T) {
	// 截图场景：字典序会把 10, 11 排在 2 前面，自然序应按数值排列
	got := []string{
		"10.冰雪奇缘·10·向北山出发.mp3",
		"11.冰雪奇缘·11·到达北山.mp3",
		"1.【试听】冰雪奇缘·01.mp3",
		"2.【试听】冰雪奇缘·02·安娜得救了.mp3",
		"3.冰雪奇缘·03·寂静的城堡.mp3",
	}
	want := []string{
		"1.【试听】冰雪奇缘·01.mp3",
		"2.【试听】冰雪奇缘·02·安娜得救了.mp3",
		"3.冰雪奇缘·03·寂静的城堡.mp3",
		"10.冰雪奇缘·10·向北山出发.mp3",
		"11.冰雪奇缘·11·到达北山.mp3",
	}

	slices.SortFunc(got, CompareNatural)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("排序结果不符:\n got = %v\nwant = %v", got, want)
		}
	}
}
