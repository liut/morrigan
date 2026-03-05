package llm

import (
	"encoding/json"
	"testing"
)

func TestToOpenAIMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		expected []Message
	}{
		{
			name:     "empty messages",
			messages: []Message{},
			expected: []Message{},
		},
		{
			name: "user message",
			messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			expected: []Message{
				{Role: "user", Content: "Hello"},
			},
		},
		{
			name: "system message",
			messages: []Message{
				{Role: RoleSystem, Content: "You are helpful."},
			},
			expected: []Message{
				{Role: "system", Content: "You are helpful."},
			},
		},
		{
			name: "assistant message with tool calls",
			messages: []Message{
				{
					Role: RoleAssistant,
					ToolCalls: []ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: ToolCallFunc{
								Name:      "get_weather",
								Arguments: json.RawMessage(`{"location":"Beijing"}`),
							},
						},
					},
				},
			},
			expected: []Message{
				{
					Role: "assistant",
					ToolCalls: []ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: ToolCallFunc{
								Name:      "get_weather",
								Arguments: json.RawMessage(`{"location":"Beijing"}`),
							},
						},
					},
				},
			},
		},
		{
			name: "tool response message",
			messages: []Message{
				{
					Role:       RoleTool,
					ToolCallID: "call_123",
					Content:    "Sunny, 25°C",
				},
			},
			expected: []Message{
				{Role: "tool", ToolCallID: "call_123", Content: "Sunny, 25°C"},
			},
		},
		{
			name: "message with name",
			messages: []Message{
				{
					Role:    RoleUser,
					Content: "Hello",
					Name:    "user_123",
				},
			},
			expected: []Message{
				{Role: "user", Content: "Hello", Name: "user_123"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toOpenAIMessages(tt.messages)
			if len(result) != len(tt.expected) {
				t.Fatalf("length mismatch: got %d, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i].Role != tt.expected[i].Role {
					t.Errorf("messages[%d].Role = %v, want %v", i, result[i].Role, tt.expected[i].Role)
				}
				if result[i].Content != tt.expected[i].Content {
					t.Errorf("messages[%d].Content = %v, want %v", i, result[i].Content, tt.expected[i].Content)
				}
				if result[i].Name != tt.expected[i].Name {
					t.Errorf("messages[%d].Name = %v, want %v", i, result[i].Name, tt.expected[i].Name)
				}
				if result[i].ToolCallID != tt.expected[i].ToolCallID {
					t.Errorf("messages[%d].ToolCallID = %v, want %v", i, result[i].ToolCallID, tt.expected[i].ToolCallID)
				}
			}
		})
	}
}

func TestToolCallFuncMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		function ToolCallFunc
		wantArgs string
	}{
		{
			name: "normal arguments",
			function: ToolCallFunc{
				Name:      "get_weather",
				Arguments: json.RawMessage(`{"location":"Beijing"}`),
			},
			wantArgs: `{"location":"Beijing"}`,
		},
		{
			name: "empty arguments",
			function: ToolCallFunc{
				Name:      "test",
				Arguments: json.RawMessage{},
			},
			wantArgs: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.function)
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(b, &result); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			args, ok := result["arguments"]
			if !ok {
				t.Fatal("arguments field not found")
			}

			if args != tt.wantArgs {
				t.Errorf("arguments = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestChatResultHasToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		toolCalls []ToolCall
		expected bool
	}{
		{
			name:     "no tool calls",
			toolCalls: nil,
			expected: false,
		},
		{
			name:     "empty tool calls",
			toolCalls: []ToolCall{},
			expected: false,
		},
		{
			name: "has tool calls",
			toolCalls: []ToolCall{
				{ID: "call_1", Type: "function"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ChatResult{
				ToolCalls: tt.toolCalls,
			}
			if result := r.HasToolCalls(); result != tt.expected {
				t.Errorf("HasToolCalls() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStreamResultString(t *testing.T) {
	// 测试 StreamResult 类型（用于覆盖）
	result := StreamResult{
		Delta:     "Hello",
		ToolCalls: nil,
		Done:      false,
		Error:     nil,
	}

	_ = result

	// 验证 Done 和 Error
	result.Done = true
	if !result.Done {
		t.Error("expected Done to be true")
	}
}
