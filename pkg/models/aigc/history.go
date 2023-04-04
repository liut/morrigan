package aigc

import (
	"encoding/json"
	"sort"
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

func (z HistoryItems) RecentlyWith(size int) (ohi HistoryItems) {
	var count int
	sort.Sort(sort.Reverse(HiAscend(z)))
	for _, hi := range z {
		count += hi.calcTokens()
		if count > size {
			break
		}
		ohi = append(ohi, hi)
	}
	sort.Sort(HiAscend(ohi))
	return
}
