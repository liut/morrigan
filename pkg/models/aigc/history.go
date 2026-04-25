package aigc

import (
	"encoding/json"
	"strings"

	"github.com/liut/morign/pkg/utils/words"
)

// HistoryChatItem is a single message in chat history
type HistoryChatItem struct {
	User      string `json:"u"`
	Assistant string `json:"a"`
	Think     string `json:"th,omitempty"` // reasoning/thinking content
}

// HistoryItem is a history record item with timestamp and chat content
type HistoryItem struct {
	Time int64 `json:"ts"`

	UID string `json:"uid,omitempty"`

	// chat
	ChatItem *HistoryChatItem `json:"ci"`
}

// calcTokens calculates the token count for history record (approximate)
func (z *HistoryItem) calcTokens() (c int) {
	if z.ChatItem != nil {
		// TODO: calculate tokens.
		c += len(z.ChatItem.User) + len(z.ChatItem.Assistant)
	}
	return
}

// HistoryItems is a list of history records
type HistoryItems []HistoryItem

// ToText 将历史记录转换为纯文本格式
func (z HistoryItems) ToText() string {
	if len(z) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, item := range z {
		if item.ChatItem != nil {
			if item.ChatItem.User != "" {
				sb.WriteString("用户: ")
				sb.WriteString(item.ChatItem.User)
				sb.WriteString("\n")
			}
			if item.ChatItem.Assistant != "" {
				sb.WriteString("助手: ")
				sb.WriteString(item.ChatItem.Assistant)
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (z *HistoryItem) MarshalBinary() (data []byte, err error) {
	data, err = json.Marshal(z)
	return
}

// UnmarshalBinary unmarshal a binary representation of itself. for redis result.Scan
func (z *HistoryItem) UnmarshalBinary(data []byte) error {
	var t HistoryItem
	err := json.Unmarshal(data, &t)
	if err == nil {
		*z = t
	}
	return err
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (z HistoryItems) MarshalBinary() (data []byte, err error) {
	data, err = json.Marshal(z)
	return
}

// UnmarshalBinary unmarshal a binary representation of itself. for redis result.Scan
func (z *HistoryItems) UnmarshalBinary(data []byte) error {
	var t HistoryItems
	err := json.Unmarshal(data, &t)
	if err == nil {
		*z = t
	}
	return err
}

// HiAscend is history records sorted by time in ascending order, for sort.Interface
type HiAscend HistoryItems

// Len returns the number of history records
func (a HiAscend) Len() int           { return len(a) }

// Swap swaps the positions of two history records
func (a HiAscend) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// Less compares the time of two history records for ascending order
func (a HiAscend) Less(i, j int) bool { return a[i].Time < a[j].Time }

// RecentlyWithTokens returns the most recent history records with total tokens not exceeding size
func (z HistoryItems) RecentlyWithTokens(size int) (ohi HistoryItems) {
	var count int
	// 从后向前遍历，直接获取最新的记录
	for i := len(z) - 1; i >= 0; i-- {
		count += z[i].calcTokens()
		if count > size {
			break
		}
		// 在开头插入元素，保持时间顺序
		ohi = append(HistoryItems{z[i]}, ohi...)
	}
	return
}

// HiLogged 是 HistoryItems 的自定义类型，用于日志输出
type HiLogged HistoryItems

// String 返回每个记录的前30个字的文本，用于日志记录
func (z HiLogged) String() string {
	if len(z) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, item := range z {
		if i > 0 {
			sb.WriteString(", ")
		}
		text := item.previewText(30)
		sb.WriteString(text)
	}
	sb.WriteString("]")
	return sb.String()
}

// previewText 返回记录的前n个字的文本
func (z *HistoryItem) previewText(n int) string {
	var text string
	if z.ChatItem != nil {
		hasUser := z.ChatItem.User != ""
		hasAssistant := z.ChatItem.Assistant != ""
		if hasUser && hasAssistant {
			text = "U: " + z.ChatItem.User + " / A: " + z.ChatItem.Assistant
		} else if hasUser {
			text = "U: " + z.ChatItem.User
		} else if hasAssistant {
			text = "A: " + z.ChatItem.Assistant
		}
	}
	return words.TakeHead(text, n, "...")
}
