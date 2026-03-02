// This file is generated - Do Not Edit.

package convo

import (
	"fmt"

	comm "github.com/cupogo/andvari/models/comm"
	oid "github.com/cupogo/andvari/models/oid"
)

func init() {
	oid.RegistCate(SessionLabel, "cs")
}

// 状态
type SessionStatus int8

const (
	SessionStatusOpen   SessionStatus = 1 + iota //  1 开启
	SessionStatusClosed                          //  2 关闭
)

func (z *SessionStatus) Decode(s string) error {
	switch s {
	case "1", "open", "Open":
		*z = SessionStatusOpen
	case "2", "closed", "Closed":
		*z = SessionStatusClosed
	default:
		return fmt.Errorf("invalid sessionStatus: %q", s)
	}
	return nil
}
func (z *SessionStatus) UnmarshalText(b []byte) error {
	return z.Decode(string(b))
}
func (z SessionStatus) String() string {
	switch z {
	case SessionStatusOpen:
		return "open"
	case SessionStatusClosed:
		return "closed"
	default:
		return fmt.Sprintf("sessionStatus %d", int8(z))
	}
}
func (z SessionStatus) MarshalText() ([]byte, error) {
	return []byte(z.String()), nil
}

// consts of Session 会话
const (
	SessionTable = "convo_session"
	SessionAlias = "cs"
	SessionLabel = "session"
	SessionTypID = "convoSession"
)

// Session 会话
type Session struct {
	comm.BaseModel `bun:"table:convo_session,alias:cs" json:"-"`

	comm.DefaultModel

	SessionBasic

	comm.MetaField

	comm.OwnerField
} // @name convoSession

type SessionBasic struct {
	// 标题
	Title string `binding:"required" bson:"title" bun:",notnull" extensions:"x-order=A" form:"title" json:"title" pg:",notnull"`
	// 状态
	//  * `open` - 开启
	//  * `closed` - 关闭
	Status SessionStatus `bson:"status" bun:",notnull,type:smallint" enums:"open,closed" extensions:"x-order=B" form:"status" json:"status" pg:",notnull,type:smallint" swaggertype:"string"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name convoSessionBasic

type Sessions []Session

// Creating function call to it's inner fields defined hooks
func (z *Session) Creating() error {
	if z.IsZeroID() {
		id, ok := oid.NewWithCode(SessionLabel)
		if !ok {
			id = oid.NewID(oid.OtEvent)
		}
		z.SetID(id)
	}

	return z.DefaultModel.Creating()
}
func NewSessionWithBasic(in SessionBasic) *Session {
	obj := &Session{
		SessionBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewSessionWithID(id any) *Session {
	obj := new(Session)
	_ = obj.SetID(id)
	return obj
}
func (_ *Session) IdentityLabel() string { return SessionLabel }
func (_ *Session) IdentityModel() string { return SessionTypID }
func (_ *Session) IdentityTable() string { return SessionTable }
func (_ *Session) IdentityAlias() string { return SessionAlias }

type SessionSet struct {
	// 标题
	Title *string `extensions:"x-order=A" json:"title"`
	// 状态
	//  * `open` - 开启
	//  * `closed` - 关闭
	Status *SessionStatus `enums:"open,closed" extensions:"x-order=B" json:"status" swaggertype:"string"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
	// 仅用于更新所有者(负责人)
	OwnerID *string `extensions:"x-order=C" json:"ownerID,omitempty"`
} // @name convoSessionSet

func (z *Session) SetWith(o SessionSet) {
	if o.Title != nil && z.Title != *o.Title {
		z.LogChangeValue("title", z.Title, o.Title)
		z.Title = *o.Title
	}
	if o.Status != nil && z.Status != *o.Status {
		z.LogChangeValue("status", z.Status, o.Status)
		z.Status = *o.Status
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
	if o.OwnerID != nil {
		if id := oid.Cast(*o.OwnerID); z.OwnerID != id {
			z.LogChangeValue("owner_id", z.OwnerID, id)
			z.SetOwnerID(id)
		}
	}
}
func (in *SessionBasic) MetaAddKVs(args ...any) *SessionBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *SessionSet) MetaAddKVs(args ...any) *SessionSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// consts of Message 会话
const (
	MessageTable = "convo_message"
	MessageAlias = "cm"
	MessageLabel = "message"
	MessageTypID = "convoMessage"
)

// Message 会话
type Message struct {
	comm.BaseModel `bun:"table:convo_message,alias:cm" json:"-"`

	comm.DefaultModel

	MessageBasic

	comm.MetaField
} // @name convoMessage

type MessageBasic struct {
	// 会话编号
	SessionID oid.OID `binding:"required" bson:"session_id" bun:",notnull" extensions:"x-order=A" json:"session" pg:",notnull" swaggertype:"string"`
	// 角色
	Role string `binding:"required" bson:"role" bun:",notnull" extensions:"x-order=B" form:"role" json:"role" pg:",notnull"`
	// 内容
	Content string `binding:"required" bson:"content" bun:",notnull" extensions:"x-order=C" form:"content" json:"content" pg:",notnull"`
	// TokenCount
	TokenCount int `bson:"tokenCount" bun:",notnull,type:smallint" extensions:"x-order=D" form:"tokenCount" json:"tokenCount" pg:",notnull,type:smallint"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name convoMessageBasic

type Messages []Message

// Creating function call to it's inner fields defined hooks
func (z *Message) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtMessage))
	}

	return z.DefaultModel.Creating()
}
func NewMessageWithBasic(in MessageBasic) *Message {
	obj := &Message{
		MessageBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewMessageWithID(id any) *Message {
	obj := new(Message)
	_ = obj.SetID(id)
	return obj
}
func (_ *Message) IdentityLabel() string { return MessageLabel }
func (_ *Message) IdentityModel() string { return MessageTypID }
func (_ *Message) IdentityTable() string { return MessageTable }
func (_ *Message) IdentityAlias() string { return MessageAlias }

type MessageSet struct {
	// 会话编号
	SessionID *string `extensions:"x-order=A" json:"session"`
	// 角色
	Role *string `extensions:"x-order=B" json:"role"`
	// 内容
	Content *string `extensions:"x-order=C" json:"content"`
	// TokenCount
	TokenCount *int `extensions:"x-order=D" json:"tokenCount"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name convoMessageSet

func (z *Message) SetWith(o MessageSet) {
	if o.SessionID != nil {
		if id := oid.Cast(*o.SessionID); z.SessionID != id {
			z.LogChangeValue("session_id", z.SessionID, id)
			z.SessionID = id
		}
	}
	if o.Role != nil && z.Role != *o.Role {
		z.LogChangeValue("role", z.Role, o.Role)
		z.Role = *o.Role
	}
	if o.Content != nil && z.Content != *o.Content {
		z.LogChangeValue("content", z.Content, o.Content)
		z.Content = *o.Content
	}
	if o.TokenCount != nil && z.TokenCount != *o.TokenCount {
		z.LogChangeValue("token_count", z.TokenCount, o.TokenCount)
		z.TokenCount = *o.TokenCount
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *MessageBasic) MetaAddKVs(args ...any) *MessageBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *MessageSet) MetaAddKVs(args ...any) *MessageSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
