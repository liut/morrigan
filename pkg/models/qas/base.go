package qas

import "strings"

const (
	VectorLen = 1024

	PrefixQ = "Q:"
	PrefixA = "A:"
)

type Vector []float32

// Pair 一对问答
type Pair struct {
	Question string `json:"Q"`
	Anwser   string `json:"A"`
}

type Pairs []Pair

func ParseText(text string) (result Pairs) {
	if len(text) < 2 {
		return
	}
	if text[0] == ' ' {
		text = PrefixQ + text
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")

	result = Pairs{}

	var lastType rune
	var lastPair Pair

	for _, line := range lines {
		if strings.HasPrefix(line, PrefixQ) {
			if lastType == 'A' {
				result = append(result, lastPair)
				lastPair = Pair{}
			}
			lastPair.Question = strings.TrimSpace(line[2:])
			lastType = 'Q'

		} else if strings.HasPrefix(line, PrefixA) {
			if lastType == 'Q' {
				lastPair.Anwser = strings.TrimSpace(line[2:])
			}
			lastType = 'A'

		} else if lastType == 'A' {
			lastPair.Anwser += "\n" + strings.TrimSpace(line)
		} else if lastType == 'Q' {
			lastPair.Question += "\n" + strings.TrimSpace(line)
		}
	}

	if lastType != 0 && lastPair.Anwser != "" {
		result = append(result, lastPair)
	}

	// 返回解析后的结果列表
	return result
}
