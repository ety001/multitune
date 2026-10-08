package fsutil

import "strings"

// CompareNatural 按自然顺序比较两个字符串：数字段按数值比较（"2" < "10"），
// 其余字符按字节比较。数值相等但写法不同（如 "1" 与 "01"）时位数少者在前。
// 用于文件名排序，使列表顺序符合 1, 2, ..., 9, 10, 11 的人类阅读直觉。
func CompareNatural(a, b string) int {
	i, j := 0, 0
	// 首个“数值相等但位数不同”的差异，仅在整个字符串其余部分完全相同时用作决胜
	tie := 0
	for i < len(a) && j < len(b) {
		ca, cb := a[i], b[j]
		if isDigit(ca) && isDigit(cb) {
			si, sj := i, j
			for i < len(a) && isDigit(a[i]) {
				i++
			}
			for j < len(b) && isDigit(b[j]) {
				j++
			}
			// 去掉前导零后，位数多者数值必然更大
			da := strings.TrimLeft(a[si:i], "0")
			db := strings.TrimLeft(b[sj:j], "0")
			if len(da) != len(db) {
				if len(da) < len(db) {
					return -1
				}
				return 1
			}
			if c := strings.Compare(da, db); c != 0 {
				return c
			}
			if tie == 0 && i-si != j-sj {
				if i-si < j-sj {
					tie = -1
				} else {
					tie = 1
				}
			}
			continue
		}
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case i < len(a):
		return 1
	case j < len(b):
		return -1
	}
	return tie
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
