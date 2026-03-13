package textutil

// TakeHead 按 rune（字符）取字符串前 n 个字符，避免截断 UTF-8 多字节字符
// ellipsis 指定省略号内容，如 "..."、"…"，为空时不添加省略号
func TakeHead(s string, n int, ellipsis ...string) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	s = string(runes[:n])
	if len(ellipsis) > 0 {
		s += ellipsis[0]
	}
	return s
}
