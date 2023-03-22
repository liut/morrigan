package conversatio

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
