package qas

const (
	VectorLen = 1536
)

type Vector []float32

// Pair 一对问答
type Pair struct {
	Qustion string `json:"Q"`
	Anwser  string `json:"A"`
}

type Pairs []Pair
