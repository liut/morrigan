package convo

import (
	"testing"
)

func TestEvaluateImportance(t *testing.T) {
	tests := []struct {
		name string
		text string
		min  float64
		max  float64
		desc string
	}{
		{
			name: "high importance - preferences with keywords",
			text: "用户偏好黑暗模式，always 使用 dark theme",
			min:  0.75,
			max:  1.0,
			desc: "strong preference + keyword → long-term candidate",
		},
		{
			name: "low importance - casual statement",
			text: "今天天气不错",
			min:  0,
			max:  0.15,
			desc: "casual → working tier",
		},
		{name: "empty string", text: "", min: 0, max: 0.001, desc: "empty → zero"},
		{
			name: "moderate importance - question",
			text: "你能记住我的生日吗？",
			min:  0.40,
			max:  0.65,
			desc: "question + keyword → moderate score",
		},
		{
			name: "urgent keyword",
			text: "urgent: 需要紧急处理的事情",
			min:  0.75,
			max:  1.0,
			desc: "urgent keyword → elevated score",
		},
		{
			name: "rule keyword",
			text: "rule: 必须遵守这个规则",
			min:  0.60,
			max:  0.80,
			desc: "multiple rule keywords → high score",
		},
		{
			name: "only punctuation no keywords",
			text: "??!!！？",
			min:  0.40,
			max:  0.65,
			desc: "punctuation only → moderate",
		},
		{
			name: "very long text should not exceed 1.0",
			text: repeat("重要且紧急的事项需要立即处理。 ", 200),
			min:  0.7,
			max:  1.0,
			desc: "long text capped at 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateImportance(tt.text)
			if got < tt.min || got > tt.max {
				t.Errorf("EvaluateImportance(%q) = %.3f, want [%.2f, %.2f] — %s",
					tt.text, got, tt.min, tt.max, tt.desc)
			}
		})
	}
}

func repeat(s string, n int) string {
	result := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		result = append(result, s...)
	}
	return string(result)
}
