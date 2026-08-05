package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/liut/morign/pkg/models/skills"
	"github.com/liut/morign/pkg/services/llm"
	"github.com/liut/morign/pkg/settings"
)

type fakeSkillStore struct {
	byName map[string]*skills.Skill
	recent []skills.Skill
	err    error
}

func (f *fakeSkillStore) TopRecent(ctx context.Context, limit int) (skills.Skills, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit <= 0 || limit > len(f.recent) {
		limit = len(f.recent)
	}
	return f.recent[:limit], nil
}

func (f *fakeSkillStore) LoadForName(ctx context.Context, name string) (*skills.Skill, error) {
	if f.err != nil {
		return nil, f.err
	}
	obj, ok := f.byName[name]
	if !ok {
		return nil, errSkillNotFound
	}
	return obj, nil
}

var errSkillNotFound = errFakeNotFound{}

type errFakeNotFound struct{}

func (errFakeNotFound) Error() string { return "skill not found" }

func TestConvertToolCallsForJSON(t *testing.T) {
	tests := []struct {
		name  string
		input []llm.ToolCall
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:  "empty slice",
			input: []llm.ToolCall{},
		},
		{
			name: "single tool call",
			input: []llm.ToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "get_weather",
						Arguments: json.RawMessage(`{"location":"Beijing"}`),
					},
				},
			},
		},
		{
			name: "multiple tool calls",
			input: []llm.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "tool1",
						Arguments: json.RawMessage(`{"a":1}`),
					},
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "tool2",
						Arguments: json.RawMessage(`{"b":2}`),
					},
				},
			},
		},
		{
			name: "empty arguments",
			input: []llm.ToolCall{
				{
					ID:   "call_empty",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "empty_tool",
						Arguments: json.RawMessage{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToolCallsForJSON(tt.input)
			// 测试序列化不报错
			jsonBytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			// nil 应该序列化为 null
			if len(tt.input) == 0 {
				if string(jsonBytes) != "null" {
					t.Errorf("expected null, got %s", jsonBytes)
				}
			} else {
				// 非空情况，确保是数组
				if jsonBytes[0] != '[' {
					t.Errorf("expected array, got %s", jsonBytes)
				}
			}
		})
	}
}

func TestConvertToolCallsForJSON_Serialize(t *testing.T) {
	// 专门测试序列化不报错
	input := []llm.ToolCall{
		{
			ID:   "call_123",
			Type: "function",
			Function: llm.ToolCallFunc{
				Name:      "get_weather",
				Arguments: json.RawMessage(`{"location":"Beijing"}`),
			},
		},
	}

	result := convertToolCallsForJSON(input)
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// 验证序列化结果可以反序列化回来
	var parsed []map[string]any
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(parsed))
	}

	if parsed[0]["id"] != "call_123" {
		t.Errorf("expected id 'call_123', got %v", parsed[0]["id"])
	}
}

func TestAppendSkillPromptDirect(t *testing.T) {
	settings.Current.SkillDirectThreshold = 3
	f := &fakeSkillStore{byName: map[string]*skills.Skill{
		"a": skillFor("a", "A", "content-a"),
		"b": skillFor("b", "B", "content-b"),
	}}
	var sb strings.Builder
	appendSkillPrompt(context.Background(), &sb, f, []string{"a", "b"})
	if !strings.Contains(sb.String(), "content-a") || !strings.Contains(sb.String(), "content-b") {
		t.Errorf("expected full contents, got %q", sb.String())
	}
}

func TestAppendSkillPromptMetadata(t *testing.T) {
	settings.Current.SkillDirectThreshold = 3
	f := &fakeSkillStore{byName: map[string]*skills.Skill{}}
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		f.byName[name] = skillFor(name, "desc-"+name, "content-"+name)
	}
	var sb strings.Builder
	appendSkillPrompt(context.Background(), &sb, f, []string{"a", "b", "c", "d", "e"})
	out := sb.String()
	if !strings.Contains(out, "desc-a") || strings.Contains(out, "content-a") {
		t.Errorf("expected metadata only, got %q", out)
	}
	if !strings.Contains(out, "skill_read") {
		t.Errorf("metadata should hint skill_read, got %q", out)
	}
}

func TestAppendSkillPromptDegrade(t *testing.T) {
	f := &fakeSkillStore{err: errFakeNotFound{}}
	var sb strings.Builder
	appendSkillPrompt(context.Background(), &sb, f, []string{"a"})
	if sb.String() != "" {
		t.Errorf("store error should degrade to empty, got %q", sb.String())
	}
}

func TestAppendSkillPromptEmpty(t *testing.T) {
	var sb strings.Builder
	appendSkillPrompt(context.Background(), &sb, &fakeSkillStore{}, nil)
	if sb.String() != "" {
		t.Errorf("no skills should inject nothing, got %q", sb.String())
	}
}

func skillFor(name, desc, content string) *skills.Skill {
	return &skills.Skill{SkillBasic: skills.SkillBasic{Name: name, Description: desc, Content: content}}
}

func TestConvertToolCallsForJSON_IncompleteJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     []llm.ToolCall
		wantPanic bool
	}{
		{
			name: "incomplete JSON - missing closing brace",
			input: []llm.ToolCall{
				{
					ID:   "call_incomplete",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "test",
						Arguments: json.RawMessage(`{"location":`),
					},
				},
			},
			wantPanic: false, // 应该不 panic，只是转成空对象
		},
		{
			name: "partial JSON key",
			input: []llm.ToolCall{
				{
					ID:   "call_partial",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "test",
						Arguments: json.RawMessage(`{"l`),
					},
				},
			},
			wantPanic: false,
		},
		{
			name: "empty raw message",
			input: []llm.ToolCall{
				{
					ID:   "call_empty",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "test",
						Arguments: json.RawMessage(``),
					},
				},
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.wantPanic {
					t.Errorf("convertToolCallsForJSON panicked: %v", r)
				}
			}()
			result := convertToolCallsForJSON(tt.input)
			// 验证序列化不报错
			_, err := json.Marshal(result)
			if err != nil {
				t.Errorf("json.Marshal failed: %v", err)
			}
		})
	}
}
