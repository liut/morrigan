package aigc

import (
	"sort"
	"testing"
)

func TestHiLogged_String(t *testing.T) {
	tests := []struct {
		name     string
		items    HistoryItems
		expected string
	}{
		{
			name:     "empty",
			items:    HistoryItems{},
			expected: "[]",
		},
		{
			name: "single item",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{User: "Hello world how are you today?"}},
			},
			expected: "[U: Hello world how are you tod...]",
		},
		{
			name: "multiple items",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{User: "First message"}},
				{ChatItem: &HistoryChatItem{User: "Second message"}},
				{ChatItem: &HistoryChatItem{User: "Third message"}},
			},
			expected: "[U: First message, U: Second message, U: Third message]",
		},
		{
			name: "empty all",
			items: HistoryItems{
				{},
			},
			expected: "[]",
		},
		{
			name: "user takes priority",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{User: "User message"}},
			},
			expected: "[U: User message]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logged := HiLogged(tt.items)
			result := logged.String()
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHistoryItem_previewText(t *testing.T) {
	tests := []struct {
		name     string
		item     HistoryItem
		n        int
		expected string
	}{
		{
			name: "normal truncate",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "This is a very long message that should be truncated"},
			},
			n:        13,
			expected: "U: This is a ...",
		},
		{
			name: "no truncate needed",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "Short"},
			},
			n:        10,
			expected: "U: Short",
		},
		{
			name: "exact length",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "1234567890"},
			},
			n:        13,
			expected: "U: 1234567890",
		},
		{
			name: "empty all",
			item: HistoryItem{},
			n:     10,
			expected: "",
		},
		{
			name: "user takes priority",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "User text"},
			},
			n:        13,
			expected: "U: User text",
		},
		{
			name: "assistant fallback",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "", Assistant: "Assistant text"},
			},
			n:        16,
			expected: "A: Assistant tex...",
		},
		{
			name: "both user and assistant",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "Hello", Assistant: "Hi there"},
			},
			n:        50,
			expected: "U: Hello / A: Hi there",
		},
		{
			name: "both truncated",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "Hello world", Assistant: "Hi there friend"},
			},
			n:        15,
			expected: "U: Hello world ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.previewText(tt.n)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHistoryItem_calcTokens(t *testing.T) {
	tests := []struct {
		name     string
		item     HistoryItem
		expected int
	}{
		{
			name:     "empty",
			item:     HistoryItem{},
			expected: 0,
		},
		{
			name: "chat item user",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "hello"},
			},
			expected: 5,
		},
		{
			name: "chat item assistant",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{Assistant: "world"},
			},
			expected: 5,
		},
		{
			name: "chat item both",
			item: HistoryItem{
				ChatItem: &HistoryChatItem{User: "hello", Assistant: "world"},
			},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.calcTokens()
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestHistoryItem_MarshalBinary(t *testing.T) {
	item := HistoryItem{
		Time: 1234567890,
		ChatItem: &HistoryChatItem{
			User: "test message",
		},
		UID: "user123",
	}

	// Marshal
	data, err := item.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Unmarshal
	var restored HistoryItem
	err = restored.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if restored.Time != item.Time || restored.UID != item.UID {
		t.Errorf("got %+v, want %+v", restored, item)
	}
}

func TestHistoryItems_MarshalBinary(t *testing.T) {
	items := HistoryItems{
		{Time: 1, ChatItem: &HistoryChatItem{User: "first"}},
		{Time: 2, ChatItem: &HistoryChatItem{User: "second"}},
		{Time: 3, ChatItem: &HistoryChatItem{User: "third"}},
	}

	// Marshal
	data, err := items.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Unmarshal
	var restored HistoryItems
	err = restored.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if len(restored) != len(items) {
		t.Errorf("got %d items, want %d", len(restored), len(items))
	}
	for i := range items {
		if restored[i].Time != items[i].Time {
			t.Errorf("item %d: got %+v, want %+v", i, restored[i], items[i])
		}
	}
}

func TestHistoryItems_RecentlyWithTokens(t *testing.T) {
	tests := []struct {
		name     string
		items    HistoryItems
		size     int
		expected int
	}{
		{
			name:     "empty",
			items:    HistoryItems{},
			size:     100,
			expected: 0,
		},
		{
			name: "single small item",
			items: HistoryItems{
				{Time: 1, ChatItem: &HistoryChatItem{User: "hi"}},
			},
			size:     100,
			expected: 1,
		},
		{
			name: "exceeds size",
			items: HistoryItems{
				{Time: 1, ChatItem: &HistoryChatItem{User: "aaaaa"}}, // 5 chars
				{Time: 2, ChatItem: &HistoryChatItem{User: "bbbbb"}}, // 5 chars
				{Time: 3, ChatItem: &HistoryChatItem{User: "ccccc"}}, // 5 chars
			},
			size:     10, // 10 chars limit
			expected: 2,  // only first 2 items (10 chars)
		},
		{
			name: "exact size",
			items: HistoryItems{
				{Time: 1, ChatItem: &HistoryChatItem{User: "aaaaa"}},
				{Time: 2, ChatItem: &HistoryChatItem{User: "bbbbb"}},
			},
			size:     10,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.items.RecentlyWithTokens(tt.size)
			if len(result) != tt.expected {
				t.Errorf("got %d items, want %d", len(result), tt.expected)
			}
			// Verify order is preserved (oldest first)
			for i := 1; i < len(result); i++ {
				if result[i-1].Time > result[i].Time {
					t.Error("items not in ascending time order")
				}
			}
		})
	}
}

func TestHiAscend(t *testing.T) {
	items := HistoryItems{
		{Time: 3, ChatItem: &HistoryChatItem{User: "third"}},
		{Time: 1, ChatItem: &HistoryChatItem{User: "first"}},
		{Time: 2, ChatItem: &HistoryChatItem{User: "second"}},
	}

	// Sort using HiAscend
	sort.Sort(HiAscend(items))

	// Verify sorted by Time ascending
	if items[0].Time != 1 || items[1].Time != 2 || items[2].Time != 3 {
		t.Errorf("items not sorted correctly: %+v", items)
	}

	// Test Less
	ha := HiAscend(items)
	if !ha.Less(0, 1) {
		t.Error("Less(0, 1) should be true")
	}
	if ha.Less(1, 0) {
		t.Error("Less(1, 0) should be false")
	}

	// Test Swap
	ha.Swap(0, 2)
	if items[0].Time != 3 || items[2].Time != 1 {
		t.Error("Swap not working correctly")
	}

	// Test Len
	if ha.Len() != 3 {
		t.Errorf("Len got %d, want 3", ha.Len())
	}
}

func TestHistoryItems_ToText(t *testing.T) {
	tests := []struct {
		name     string
		items    HistoryItems
		expected string
	}{
		{
			name:     "empty",
			items:    HistoryItems{},
			expected: "",
		},
		{
			name: "only user message",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{User: "Hello"}},
			},
			expected: "用户: Hello\n",
		},
		{
			name: "only assistant message",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{Assistant: "Hi there"}},
			},
			expected: "助手: Hi there\n",
		},
		{
			name: "user and assistant",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{User: "Hello", Assistant: "Hi there"}},
			},
			expected: "用户: Hello\n助手: Hi there\n",
		},
		{
			name: "multiple messages",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{User: "First question"}},
				{ChatItem: &HistoryChatItem{Assistant: "First answer"}},
				{ChatItem: &HistoryChatItem{User: "Second question"}},
			},
			expected: "用户: First question\n助手: First answer\n用户: Second question\n",
		},
		{
			name: "nil chat item",
			items: HistoryItems{
				{},
			},
			expected: "",
		},
		{
			name: "empty chat item",
			items: HistoryItems{
				{ChatItem: &HistoryChatItem{}},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.items.ToText()
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
