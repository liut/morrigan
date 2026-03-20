package words

import (
	"testing"
)

func TestTakeHead(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		n        int
		ellipsis []string
		expected string
	}{
		{
			name:     "short string returns original",
			s:        "hello",
			n:        10,
			expected: "hello",
		},
		{
			name:     "exact length returns original",
			s:        "hello",
			n:        5,
			expected: "hello",
		},
		{
			name:     "truncate with default ellipsis",
			s:        "hello world",
			n:        5,
			ellipsis: []string{"..."},
			expected: "hello...",
		},
		{
			name:     "truncate with unicode ellipsis",
			s:        "hello world",
			n:        5,
			ellipsis: []string{"…"},
			expected: "hello…",
		},
		{
			name:     "truncate without ellipsis",
			s:        "hello world",
			n:        5,
			expected: "hello",
		},
		{
			name:     "multibyte characters",
			s:        "你好世界",
			n:        2,
			ellipsis: []string{"..."},
			expected: "你好...",
		},
		{
			name:     "mixed multibyte and ascii",
			s:        "hello你好",
			n:        5,
			ellipsis: []string{"..."},
			expected: "hello...",
		},
		{
			name:     "empty string",
			s:        "",
			n:        5,
			expected: "",
		},
		{
			name:     "zero n returns empty",
			s:        "hello",
			n:        0,
			expected: "",
		},
		{
			name:     "negative n returns empty",
			s:        "hello",
			n:        -1,
			expected: "",
		},
		{
			name:     "empty ellipsis no effect",
			s:        "hello world",
			n:        5,
			ellipsis: []string{""},
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if len(tt.ellipsis) > 0 {
				result = TakeHead(tt.s, tt.n, tt.ellipsis[0])
			} else {
				result = TakeHead(tt.s, tt.n)
			}
			if result != tt.expected {
				t.Errorf("TakeHead(%q, %d, %v) = %q, want %q", tt.s, tt.n, tt.ellipsis, result, tt.expected)
			}
		})
	}
}

func TestTakeTail(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		n        int
		ellipsis []string
		expected string
	}{
		{
			name:     "short string returns original",
			s:        "hello",
			n:        10,
			expected: "hello",
		},
		{
			name:     "exact length returns original",
			s:        "hello",
			n:        5,
			expected: "hello",
		},
		{
			name:     "truncate with default ellipsis",
			s:        "hello world",
			n:        5,
			ellipsis: []string{"..."},
			expected: "...world",
		},
		{
			name:     "truncate with unicode ellipsis",
			s:        "hello world",
			n:        5,
			ellipsis: []string{"…"},
			expected: "…world",
		},
		{
			name:     "truncate without ellipsis",
			s:        "hello world",
			n:        5,
			expected: "world",
		},
		{
			name:     "multibyte characters",
			s:        "你好世界",
			n:        2,
			ellipsis: []string{"..."},
			expected: "...世界",
		},
		{
			name:     "mixed multibyte and ascii",
			s:        "hello你好",
			n:        3,
			ellipsis: []string{"..."},
			expected: "...o你好",
		},
		{
			name:     "empty string",
			s:        "",
			n:        5,
			expected: "",
		},
		{
			name:     "zero n returns empty",
			s:        "hello",
			n:        0,
			expected: "",
		},
		{
			name:     "negative n returns empty",
			s:        "hello",
			n:        -1,
			expected: "",
		},
		{
			name:     "empty ellipsis no effect",
			s:        "hello world",
			n:        5,
			ellipsis: []string{""},
			expected: "world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if len(tt.ellipsis) > 0 {
				result = TakeTail(tt.s, tt.n, tt.ellipsis[0])
			} else {
				result = TakeTail(tt.s, tt.n)
			}
			if result != tt.expected {
				t.Errorf("TakeTail(%q, %d, %v) = %q, want %q", tt.s, tt.n, tt.ellipsis, result, tt.expected)
			}
		})
	}
}
