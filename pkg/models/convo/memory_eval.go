package convo

import (
	"math"
	"strings"
)

// importanceKeywords maps keywords to weight scores for importance evaluation.
var importanceKeywords = map[string]float64{
	// English
	"preference": 0.4,
	"always":     0.3,
	"never":      0.3,
	"urgent":     0.4,
	"important":  0.3,
	"remember":   0.25,
	"password":   0.3,
	"secret":     0.3,
	"rule":       0.2,
	"must":       0.2,
	// Chinese
	"偏好":   0.4,
	"总是":   0.3,
	"从不":   0.3,
	"紧急":   0.4,
	"重要":   0.3,
	"记住":   0.35,
	"密码":   0.3,
	"规则":   0.2,
	"必须":   0.2,
}

const maxLogLen = 500.0

// EvaluateImportance scores a memory text for importance (0.0–1.0)
// using rule-based heuristics: length, punctuation, and keyword matching.
func EvaluateImportance(text string) float64 {
	if text == "" {
		return 0
	}

	runes := []rune(text)
	charLen := float64(len(runes))

	lenScore := math.Log(charLen+1) / math.Log(maxLogLen)
	if lenScore > 1.0 {
		lenScore = 1.0
	}

	punctCount := strings.Count(text, "?") + strings.Count(text, "!") +
		strings.Count(text, "？") + strings.Count(text, "！")
	punctScore := float64(punctCount) * 0.08

	textLower := strings.ToLower(text)
	var kwScore float64
	for kw, weight := range importanceKeywords {
		if strings.Contains(textLower, kw) {
			kwScore += weight
		}
	}
	if kwScore > 0.85 {
		kwScore = 0.85
	}

	return clamp01(lenScore*0.2 + punctScore + kwScore)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}
