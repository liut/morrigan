// This file is generated - Do Not Edit.

package qas

import (
	comm "github.com/cupogo/andvari/models/comm"
	oid "github.com/cupogo/andvari/models/oid"
)

// consts of Document 文档
const (
	DocumentTable = "qa_corpus_document"
	DocumentAlias = "cd"
	DocumentLabel = "document"
)

// Document 文档 语料库
type Document struct {
	comm.BaseModel `bun:"table:qa_corpus_document,alias:cd" json:"-"`

	comm.DefaultModel

	DocumentBasic

	comm.MetaField
} // @name qasDocument

type DocumentBasic struct {
	// 主标题
	Title string `bun:",notnull,type:text,unique:corpus_title_heading_key" extensions:"x-order=A" form:"title" json:"title" pg:",notnull,type:text,unique:corpus_title_heading_key"`
	// 小节标题
	Heading string `bun:",notnull,type:text,unique:corpus_title_heading_key" extensions:"x-order=B" form:"heading" json:"heading" pg:",notnull,type:text,unique:corpus_title_heading_key"`
	// 内容
	Content string `bun:",notnull,type:text" extensions:"x-order=C" form:"content" json:"content" pg:",notnull,type:text"`
	// 问答集
	QAs Pairs `bun:"qas,notnull,type:jsonb" extensions:"x-order=D" json:"qas,omitempty" pg:"qas,notnull,type:jsonb"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name qasDocumentBasic

type Documents []Document

// Creating function call to it's inner fields defined hooks
func (z *Document) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtArticle))
	}

	return z.DefaultModel.Creating()
}
func NewDocumentWithBasic(in DocumentBasic) *Document {
	obj := &Document{
		DocumentBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewDocumentWithID(id any) *Document {
	obj := new(Document)
	_ = obj.SetID(id)
	return obj
}
func (_ *Document) IdentityLabel() string {
	return DocumentLabel
}
func (_ *Document) IdentityTable() string {
	return DocumentTable
}
func (_ *Document) IdentityAlias() string {
	return DocumentAlias
}

type DocumentSet struct {
	// 主标题
	Title *string `extensions:"x-order=A" json:"title"`
	// 小节标题
	Heading *string `extensions:"x-order=B" json:"heading"`
	// 内容
	Content *string `extensions:"x-order=C" json:"content"`
	// 问答集
	QAs *Pairs `extensions:"x-order=D" json:"qas,omitempty"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name qasDocumentSet

func (z *Document) SetWith(o DocumentSet) {
	if o.Title != nil && z.Title != *o.Title {
		z.LogChangeValue("title", z.Title, o.Title)
		z.Title = *o.Title
	}
	if o.Heading != nil && z.Heading != *o.Heading {
		z.LogChangeValue("heading", z.Heading, o.Heading)
		z.Heading = *o.Heading
	}
	if o.Content != nil && z.Content != *o.Content {
		z.LogChangeValue("content", z.Content, o.Content)
		z.Content = *o.Content
	}
	if o.QAs != nil {
		z.LogChangeValue("qas", z.QAs, o.QAs)
		z.QAs = *o.QAs
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *DocumentBasic) MetaAddKVs(args ...any) *DocumentBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *DocumentSet) MetaAddKVs(args ...any) *DocumentSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// PromptMatch 提示匹配结果
type PromptMatch struct {
	// 文档编号
	DocID oid.OID `extensions:"x-order=A" json:"docID"`
	// 提示
	Prompt string `extensions:"x-order=B" form:"prompt" json:"prompt"`
	// 相似度
	Similarity float32 `extensions:"x-order=C" json:"similarity,omitempty"`
} // @name qasPromptMatch

type PromptMatches []PromptMatch

// consts of Prompt 提示及向量
const (
	PromptTable = "qa_corpus_prompt"
	PromptAlias = "cp"
	PromptLabel = "prompt"
)

// Prompt 提示及向量
type Prompt struct {
	comm.BaseModel `bun:"table:qa_corpus_prompt,alias:cp" json:"-"`

	comm.DefaultModel

	PromptBasic

	// 相似度 仅用于查询结果
	Similarity float32 `bun:",notnull,type:float4" extensions:"x-order=E" json:"similarity,omitempty" pg:",notnull,type:float4"`

	comm.MetaField
} // @name qasPrompt

type PromptBasic struct {
	// 文档编号
	DocID oid.OID `bun:"doc_id,notnull" extensions:"x-order=A" json:"docID" pg:"doc_id,notnull"`
	// 提示
	Text string `bun:"prompt,notnull,type:text,unique" extensions:"x-order=B" form:"prompt" json:"prompt" pg:"prompt,notnull,type:text,unique"`
	// Tokens
	Tokens uint `bun:",notnull,type:smallint" extensions:"x-order=C" form:"tokens" json:"tokens,omitempty" pg:",notnull,type:smallint"`
	// 向量值 长为1536的浮点数集
	Vector Vector `bun:"embedding,type:vector(1536)" extensions:"x-order=D" json:"vector,omitempty" pg:"embedding,type:vector(1536)"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name qasPromptBasic

type Prompts []Prompt

// Creating function call to it's inner fields defined hooks
func (z *Prompt) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtArticle))
	}

	return z.DefaultModel.Creating()
}
func NewPromptWithBasic(in PromptBasic) *Prompt {
	obj := &Prompt{
		PromptBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewPromptWithID(id any) *Prompt {
	obj := new(Prompt)
	_ = obj.SetID(id)
	return obj
}
func (_ *Prompt) IdentityLabel() string {
	return PromptLabel
}
func (_ *Prompt) IdentityTable() string {
	return PromptTable
}
func (_ *Prompt) IdentityAlias() string {
	return PromptAlias
}

type PromptSet struct {
	// 文档编号
	DocID *string `extensions:"x-order=A" json:"docID"`
	// 提示
	Text *string `extensions:"x-order=B" json:"prompt"`
	// Tokens
	Tokens *uint `extensions:"x-order=C" json:"tokens,omitempty"`
	// 向量值 长为1536的浮点数集
	Vector *Vector `extensions:"x-order=D" json:"vector,omitempty"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name qasPromptSet

func (z *Prompt) SetWith(o PromptSet) {
	if o.DocID != nil {
		if id := oid.Cast(*o.DocID); z.DocID != id {
			z.LogChangeValue("doc_id", z.DocID, id)
			z.DocID = id
		}
	}
	if o.Text != nil && z.Text != *o.Text {
		z.LogChangeValue("prompt", z.Text, o.Text)
		z.Text = *o.Text
	}
	if o.Tokens != nil && z.Tokens != *o.Tokens {
		z.LogChangeValue("tokens", z.Tokens, o.Tokens)
		z.Tokens = *o.Tokens
	}
	if o.Vector != nil {
		z.LogChangeValue("embedding", z.Vector, o.Vector)
		z.Vector = *o.Vector
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *PromptBasic) MetaAddKVs(args ...any) *PromptBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *PromptSet) MetaAddKVs(args ...any) *PromptSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// consts of ChatLog 聊天日志
const (
	ChatLogTable = "qa_chat_log"
	ChatLogAlias = "cl"
	ChatLogLabel = "chatLog"
)

// ChatLog 聊天日志
type ChatLog struct {
	comm.BaseModel `bun:"table:qa_chat_log,alias:cl" json:"-"`

	comm.DefaultModel

	ChatLogBasic

	comm.MetaField
} // @name qasChatLog

type ChatLogBasic struct {
	// 会话ID
	ChatID oid.OID `bun:"csid,notnull" extensions:"x-order=A" json:"csid" pg:"csid,notnull"`
	// 提问
	Question string `bun:",notnull,type:text" extensions:"x-order=B" form:"prompt" json:"prompt" pg:",notnull,type:text"`
	// 回答
	Answer string `bun:",notnull,type:text" extensions:"x-order=C" form:"response" json:"response" pg:",notnull,type:text"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name qasChatLogBasic

type ChatLogs []ChatLog

// Creating function call to it's inner fields defined hooks
func (z *ChatLog) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtEvent))
	}

	return z.DefaultModel.Creating()
}
func NewChatLogWithBasic(in ChatLogBasic) *ChatLog {
	obj := &ChatLog{
		ChatLogBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewChatLogWithID(id any) *ChatLog {
	obj := new(ChatLog)
	_ = obj.SetID(id)
	return obj
}
func (_ *ChatLog) IdentityLabel() string {
	return ChatLogLabel
}
func (_ *ChatLog) IdentityTable() string {
	return ChatLogTable
}
func (_ *ChatLog) IdentityAlias() string {
	return ChatLogAlias
}

type ChatLogSet struct {
	// 会话ID
	ChatID *string `extensions:"x-order=A" json:"csid"`
	// 提问
	Question *string `extensions:"x-order=B" json:"prompt"`
	// 回答
	Answer *string `extensions:"x-order=C" json:"response"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name qasChatLogSet

func (z *ChatLog) SetWith(o ChatLogSet) {
	if o.ChatID != nil {
		if id := oid.Cast(*o.ChatID); z.ChatID != id {
			z.LogChangeValue("csid", z.ChatID, id)
			z.ChatID = id
		}
	}
	if o.Question != nil && z.Question != *o.Question {
		z.LogChangeValue("question", z.Question, o.Question)
		z.Question = *o.Question
	}
	if o.Answer != nil && z.Answer != *o.Answer {
		z.LogChangeValue("answer", z.Answer, o.Answer)
		z.Answer = *o.Answer
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *ChatLogBasic) MetaAddKVs(args ...any) *ChatLogBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *ChatLogSet) MetaAddKVs(args ...any) *ChatLogSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
