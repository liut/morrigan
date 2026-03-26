package channel

// ImageAttachment represents an image sent by the user.
type ImageAttachment struct {
	MimeType string // e.g. "image/png", "image/jpeg"
	Data     []byte // raw image bytes
	FileName string // original filename (optional)
}

// FileAttachment represents a file (PDF, doc, spreadsheet, etc.) sent by the user.
type FileAttachment struct {
	MimeType string // e.g. "application/pdf", "text/plain"
	Data     []byte // raw file bytes
	FileName string
}

// AudioAttachment represents a voice/audio message sent by the user.
type AudioAttachment struct {
	MimeType string // e.g. "audio/amr", "audio/ogg", "audio/mp4"
	Data     []byte // raw audio bytes
	Format   string // short format hint: "amr", "ogg", "m4a", "mp3", "wav", etc.
	Duration int    // duration in seconds (if known)
}

// Message represents a unified incoming message from any channel.
type Message struct {
	SessionKey string // unique key for user context, e.g. "wecom:{userID}"
	Channel    string
	MessageID  string // channel message ID for tracing/dedup
	UserID     string
	UserName   string
	ChatName   string // human-readable chat/group name (optional)
	Content    string
	Images     []ImageAttachment // attached images (if any)
	Files      []FileAttachment  // attached files (if any)
	Audio      *AudioAttachment  // voice message (if any)
	ReplyCtx   any               // channel-specific context needed for replying
	FromVoice  bool              // true if message originated from voice transcription
}
