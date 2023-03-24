package aigc

import "encoding/json"

type HistoryChatItem struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
}

type HistoryItem struct {
	Time int64 `json:"time"`

	// text with stop mark
	Text string `json:"text,omitempty"`
	UID  string `json:"uid"`

	// chat
	ChatItem *HistoryChatItem `json:"chat"`
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
