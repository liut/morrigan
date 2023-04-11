package qas

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	text := `Q: What is your name?
A: My name is Alice.

Q: What is your favorite color?

A: My favorite color is blue.

Q: How old are you?
A: I am
25 years old.`

	result := ParseText(text)

	assert.NotEmpty(t, result)
	assert.Len(t, result, 3)

	for i, qa := range result {
		t.Logf("QA %d:", i+1)
		t.Logf("Question: %s", qa.Question)
		t.Logf("Answer:   %s", qa.Anwser)
	}

}
