package aigc

import (
	"encoding/json"
	"strings"
)

type HistoryChatItem struct {
	User      string `json:"u"`
	Assistant string `json:"a"`
}

type HistoryItem struct {
	Time int64 `json:"ts"`

	// text with stop mark
	Text string `json:"txt,omitempty"`
	UID  string `json:"uid,omitempty"`

	// chat
	ChatItem *HistoryChatItem `json:"ci"`
}

func (z *HistoryItem) calcTokens() (c int) {
	if z.ChatItem != nil {
		// TODO: calculate tokens.
		c += len(z.ChatItem.User) + len(z.ChatItem.Assistant)
	}
	c += len(z.Text)
	return
}

type HistoryItems []HistoryItem

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

type HiAscend HistoryItems

func (a HiAscend) Len() int           { return len(a) }
func (a HiAscend) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a HiAscend) Less(i, j int) bool { return a[i].Time < a[j].Time }

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
	if text == "" {
		text = z.Text
	}
	// 按 rune（字符）截取前 n 个字，避免截断 UTF-8 多字节字符
	runes := []rune(text)
	if len(runes) > n {
		return string(runes[:n]) + "..."
	}
	return text
}
