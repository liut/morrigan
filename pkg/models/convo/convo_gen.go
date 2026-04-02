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
	Title string `bson:"title" bun:",notnull" extensions:"x-order=A" form:"title" json:"title" pg:",notnull"`
	// 消息数
	MessageCount int `bson:"msgCount" bun:"msg_count,notnull,type:smallint" extensions:"x-order=B" form:"msgCount" json:"msgCount" pg:"msg_count,notnull,type:smallint"`
	// 状态
	//  * `open` - 开启
	//  * `closed` - 关闭
	Status SessionStatus `bson:"status" bun:",notnull,type:smallint" enums:"open,closed" extensions:"x-order=C" form:"status" json:"status" pg:",notnull,type:smallint" swaggertype:"string"`
	// 工具
	Tools []string `bson:"tools" bun:",notnull,default:'[]'" extensions:"x-order=D" json:"tools" pg:",notnull,default:'[]'"`
	// 频道
	Channel string `bson:"channel" bun:",notnull,type:varchat(23)" extensions:"x-order=E" form:"channel" json:"channel" pg:",notnull,type:varchat(23)"`
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
	// 消息数
	MessageCount *int `extensions:"x-order=B" json:"msgCount"`
	// 状态
	//  * `open` - 开启
	//  * `closed` - 关闭
	Status *SessionStatus `enums:"open,closed" extensions:"x-order=C" json:"status" swaggertype:"string"`
	// 工具
	Tools *[]string `extensions:"x-order=D" json:"tools"`
	// 频道
	Channel *string `extensions:"x-order=E" json:"channel"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
	// 仅用于更新所有者(负责人)
	OwnerID *string `extensions:"x-order=F" json:"ownerID,omitempty"`
} // @name convoSessionSet

func (z *Session) SetWith(o SessionSet) {
	if o.Title != nil && z.Title != *o.Title {
		z.LogChangeValue("title", z.Title, o.Title)
		z.Title = *o.Title
	}
	if o.MessageCount != nil && z.MessageCount != *o.MessageCount {
		z.LogChangeValue("msg_count", z.MessageCount, o.MessageCount)
		z.MessageCount = *o.MessageCount
	}
	if o.Status != nil && z.Status != *o.Status {
		z.LogChangeValue("status", z.Status, o.Status)
		z.Status = *o.Status
	}
	if o.Tools != nil {
		z.LogChangeValue("tools", z.Tools, o.Tools)
		z.Tools = *o.Tools
	}
	if o.Channel != nil && z.Channel != *o.Channel {
		z.LogChangeValue("channel", z.Channel, o.Channel)
		z.Channel = *o.Channel
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

// consts of Message 消息
const (
	MessageTable = "convo_message"
	MessageAlias = "cm"
	MessageLabel = "message"
	MessageTypID = "convoMessage"
)

// Message 消息
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

// consts of UsageRecord 使用情况
const (
	UsageRecordTable = "convo_usage_record"
	UsageRecordAlias = "ur"
	UsageRecordLabel = "usageRecord"
	UsageRecordTypID = "convoUsageRecord"
)

// UsageRecord 使用情况 每次请求后记录
type UsageRecord struct {
	comm.BaseModel `bun:"table:convo_usage_record,alias:ur" json:"-"`

	comm.DefaultModel

	UsageRecordBasic

	comm.MetaField
} // @name convoUsageRecord

type UsageRecordBasic struct {
	// 会话编号
	SessionID oid.OID `binding:"required" bson:"session_id" bun:",notnull" extensions:"x-order=A" json:"session" pg:",notnull" swaggertype:"string"`
	// 消息数
	MsgCount int `bson:"msgCount" bun:",notnull,type:smallint" extensions:"x-order=B" form:"msgCount" json:"msgCount" pg:",notnull,type:smallint"`
	// 输入Token数
	InputTokens int `bson:"inputTokens" bun:",notnull,type:int" extensions:"x-order=C" form:"inputTokens" json:"inputTokens" pg:",notnull,type:int"`
	// 输出Token数
	OutputTokens int `bson:"outputTokens" bun:",notnull,type:int" extensions:"x-order=D" form:"outputTokens" json:"outputTokens" pg:",notnull,type:int"`
	// 总Token数
	TotalTokens int `bson:"totalTokens" bun:",notnull,type:int" extensions:"x-order=E" form:"totalTokens" json:"totalTokens" pg:",notnull,type:int"`
	// 模型
	Model string `bson:"model" bun:",notnull,type:name" extensions:"x-order=F" form:"model" json:"model" pg:",notnull,type:name"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name convoUsageRecordBasic

type UsageRecords []UsageRecord

// Creating function call to it's inner fields defined hooks
func (z *UsageRecord) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtEvent))
	}

	return z.DefaultModel.Creating()
}
func NewUsageRecordWithBasic(in UsageRecordBasic) *UsageRecord {
	obj := &UsageRecord{
		UsageRecordBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewUsageRecordWithID(id any) *UsageRecord {
	obj := new(UsageRecord)
	_ = obj.SetID(id)
	return obj
}
func (_ *UsageRecord) IdentityLabel() string { return UsageRecordLabel }
func (_ *UsageRecord) IdentityModel() string { return UsageRecordTypID }
func (_ *UsageRecord) IdentityTable() string { return UsageRecordTable }
func (_ *UsageRecord) IdentityAlias() string { return UsageRecordAlias }

type UsageRecordSet struct {
	// 消息数
	MsgCount *int `extensions:"x-order=A" json:"msgCount"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name convoUsageRecordSet

func (z *UsageRecord) SetWith(o UsageRecordSet) {
	if o.MsgCount != nil && z.MsgCount != *o.MsgCount {
		z.LogChangeValue("msg_count", z.MsgCount, o.MsgCount)
		z.MsgCount = *o.MsgCount
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *UsageRecordBasic) MetaAddKVs(args ...any) *UsageRecordBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *UsageRecordSet) MetaAddKVs(args ...any) *UsageRecordSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// consts of User 用户
const (
	UserTable = "convo_user"
	UserAlias = "u"
	UserLabel = "user"
	UserTypID = "convoUser"
)

// User 用户 来自 OAuth SP 的拷贝
type User struct {
	comm.BaseModel `bun:"table:convo_user,alias:u" json:"-"`

	comm.DefaultModel

	UserBasic

	comm.MetaField
} // @name convoUser

type UserBasic struct {
	// 登录名 唯一
	Username string `bun:"username,notnull,type:name,unique" extensions:"x-order=A" form:"username" json:"username" pg:"username,notnull,type:name,unique"`
	// 昵称
	Nickname string `bun:"nickname,notnull,type:varchar(45)" extensions:"x-order=B" form:"nickname" json:"nickname" pg:"nickname,notnull,type:varchar(45)"`
	// 头像路径
	AvatarPath string `bun:"avatar,notnull,type:varchar(125)" extensions:"x-order=C" form:"avatar" json:"avatar,omitempty" pg:"avatar,notnull,type:varchar(125)"`
	// 邮箱
	Email string `bun:"email,notnull,type:varchar(43)" extensions:"x-order=D" form:"email" json:"email,omitempty" pg:"email,notnull,use_zero,type:varchar(43)"`
	// 电话
	Phone string `bun:"phone,notnull,type:varchar(15)" extensions:"x-order=E" form:"phone" json:"phone,omitempty" pg:"phone,notnull,use_zero,type:varchar(15)"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name convoUserBasic

type Users []User

// Creating function call to it's inner fields defined hooks
func (z *User) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtAccount))
	}

	return z.DefaultModel.Creating()
}
func NewUserWithBasic(in UserBasic) *User {
	obj := &User{
		UserBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewUserWithID(id any) *User {
	obj := new(User)
	_ = obj.SetID(id)
	return obj
}
func (_ *User) IdentityLabel() string { return UserLabel }
func (_ *User) IdentityModel() string { return UserTypID }
func (_ *User) IdentityTable() string { return UserTable }
func (_ *User) IdentityAlias() string { return UserAlias }

type UserSet struct {
	// 登录名 唯一
	Username *string `extensions:"x-order=A" json:"username"`
	// 昵称
	Nickname *string `extensions:"x-order=B" json:"nickname"`
	// 头像路径
	AvatarPath *string `extensions:"x-order=C" form:"avatar" json:"avatar,omitempty"`
	// 邮箱
	Email *string `extensions:"x-order=D" form:"email" json:"email,omitempty"`
	// 电话
	Phone *string `extensions:"x-order=E" form:"phone" json:"phone,omitempty"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name convoUserSet

func (z *User) SetWith(o UserSet) {
	if o.Username != nil && z.Username != *o.Username {
		z.LogChangeValue("username", z.Username, o.Username)
		z.Username = *o.Username
	}
	if o.Nickname != nil && z.Nickname != *o.Nickname {
		z.LogChangeValue("nickname", z.Nickname, o.Nickname)
		z.Nickname = *o.Nickname
	}
	if o.AvatarPath != nil && z.AvatarPath != *o.AvatarPath {
		z.LogChangeValue("avatar", z.AvatarPath, o.AvatarPath)
		z.AvatarPath = *o.AvatarPath
	}
	if o.Email != nil && z.Email != *o.Email {
		z.LogChangeValue("email", z.Email, o.Email)
		z.Email = *o.Email
	}
	if o.Phone != nil && z.Phone != *o.Phone {
		z.LogChangeValue("phone", z.Phone, o.Phone)
		z.Phone = *o.Phone
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *UserBasic) MetaAddKVs(args ...any) *UserBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *UserSet) MetaAddKVs(args ...any) *UserSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// consts of ThirdUser 第三方用户
const (
	ThirdUserTable = "convo_third_user"
	ThirdUserAlias = "tu"
	ThirdUserLabel = "thirdUser"
	ThirdUserTypID = "convoThirdUser"
)

// ThirdUser 第三方用户 来自不同平台 如微信和飞书等 PK为第三方平台标识+账号
type ThirdUser struct {
	comm.BaseModel `bun:"table:convo_third_user,alias:tu" json:"-"`

	comm.DunceModel

	ThirdUserBasic

	comm.MetaField
} // @name convoThirdUser

type ThirdUserBasic struct {
	// 所有人编号
	OwnerID oid.OID `bun:"owner_id,notnull,type:bigint" extensions:"x-order=A" json:"ownerID" pg:"owner_id,notnull,type:bigint" swaggertype:"string"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name convoThirdUserBasic

type ThirdUsers []ThirdUser

// Creating function call to it's inner fields defined hooks
func (z *ThirdUser) Creating() error {
	if z.IsZeroID() {
		return comm.ErrEmptyID
	}

	return z.DunceModel.Creating()
}
func NewThirdUserWithBasic(in ThirdUserBasic) *ThirdUser {
	obj := &ThirdUser{
		ThirdUserBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewThirdUserWithID(id any) *ThirdUser {
	obj := new(ThirdUser)
	_ = obj.SetID(id)
	return obj
}
func (_ *ThirdUser) IdentityLabel() string { return ThirdUserLabel }
func (_ *ThirdUser) IdentityModel() string { return ThirdUserTypID }
func (_ *ThirdUser) IdentityTable() string { return ThirdUserTable }
func (_ *ThirdUser) IdentityAlias() string { return ThirdUserAlias }

type ThirdUserSet struct {
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name convoThirdUserSet

func (z *ThirdUser) SetWith(o ThirdUserSet) {
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *ThirdUserBasic) MetaAddKVs(args ...any) *ThirdUserBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *ThirdUserSet) MetaAddKVs(args ...any) *ThirdUserSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// consts of Memory 记忆
const (
	MemoryTable = "convo_memory"
	MemoryAlias = "m"
	MemoryLabel = "memory"
	MemoryTypID = "convoMemory"
)

// Memory 记忆
type Memory struct {
	comm.BaseModel `bun:"table:convo_memory,alias:m" json:"-"`

	comm.DefaultModel

	MemoryBasic

	comm.MetaField
} // @name convoMemory

type MemoryBasic struct {
	// 所有人编号
	OwnerID oid.OID `bun:"owner_id,notnull,type:bigint,unique:mm_key_uid_key" extensions:"x-order=A" json:"ownerID" pg:"owner_id,notnull,type:bigint,unique:mm_key_uid_key" swaggertype:"string"`
	// 关键点
	Key string `bun:",notnull,type:text,unique:mm_key_uid_key" extensions:"x-order=B" form:"key" json:"key" pg:",notnull,type:text,unique:mm_key_uid_key"`
	// 分类
	Cate string `bun:",notnull,type:text" extensions:"x-order=C" form:"cate" json:"cate" pg:",notnull,type:text"`
	// 内容
	Content string `bun:",notnull,type:text" extensions:"x-order=D" form:"content" json:"content" pg:",notnull,type:text"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name convoMemoryBasic

type Memories []Memory

// Creating function call to it's inner fields defined hooks
func (z *Memory) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtEvent))
	}

	return z.DefaultModel.Creating()
}
func NewMemoryWithBasic(in MemoryBasic) *Memory {
	obj := &Memory{
		MemoryBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewMemoryWithID(id any) *Memory {
	obj := new(Memory)
	_ = obj.SetID(id)
	return obj
}
func (_ *Memory) IdentityLabel() string { return MemoryLabel }
func (_ *Memory) IdentityModel() string { return MemoryTypID }
func (_ *Memory) IdentityTable() string { return MemoryTable }
func (_ *Memory) IdentityAlias() string { return MemoryAlias }

type MemorySet struct {
	// 分类
	Cate *string `extensions:"x-order=A" json:"cate"`
	// 内容
	Content *string `extensions:"x-order=B" json:"content"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name convoMemorySet

func (z *Memory) SetWith(o MemorySet) {
	if o.Cate != nil && z.Cate != *o.Cate {
		z.LogChangeValue("cate", z.Cate, o.Cate)
		z.Cate = *o.Cate
	}
	if o.Content != nil && z.Content != *o.Content {
		z.LogChangeValue("content", z.Content, o.Content)
		z.Content = *o.Content
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *MemoryBasic) MetaAddKVs(args ...any) *MemoryBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *MemorySet) MetaAddKVs(args ...any) *MemorySet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
