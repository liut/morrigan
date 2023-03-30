// This file is generated - Do Not Edit.

package qas

import (
	comm "github.com/cupogo/andvari/models/comm"
	oid "github.com/cupogo/andvari/models/oid"
)

// consts of Document 文档
const (
	DocumentTable = "qa_corpus_document"
	DocumentAlias = "c"
	DocumentLabel = "document"
)

// Document 文档 语料库
type Document struct {
	comm.BaseModel `bun:"table:qa_corpus_document,alias:c" json:"-"`

	comm.DefaultModel

	DocumentBasic

	// 相似度 仅用于查询结果
	Similarity float32 `bun:",notnull,type:float4" extensions:"x-order=F" json:"similarity,omitempty" pg:",notnull,type:float4"`

	comm.MetaField
} // @name qasDocument

type DocumentBasic struct {
	// 主标题
	Title string `bun:",notnull,type:text,unique:corpus_title_heading_key" extensions:"x-order=A" form:"title" json:"title" pg:",notnull,type:text,unique:corpus_title_heading_key"`
	// 小节标题
	Heading string `bun:",notnull,type:text,unique:corpus_title_heading_key" extensions:"x-order=B" form:"heading" json:"heading" pg:",notnull,type:text,unique:corpus_title_heading_key"`
	// 内容
	Content string `bun:",notnull,type:text" extensions:"x-order=C" form:"content" json:"content" pg:",notnull,type:text"`
	// Tokens
	Tokens uint `bun:",notnull,type:smallint" extensions:"x-order=D" form:"tokens" json:"tokens,omitempty" pg:",notnull,type:smallint"`
	// 向量值 长为1536的浮点数集
	Embedding Vector `bun:",type:vector(1536)" extensions:"x-order=E" json:"embedding,omitempty" pg:",type:vector(1536)"`
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
	// Tokens
	Tokens *uint `extensions:"x-order=D" json:"tokens,omitempty"`
	// 向量值 长为1536的浮点数集
	Embedding *Vector `extensions:"x-order=E" json:"embedding,omitempty"`
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
	if o.Tokens != nil && z.Tokens != *o.Tokens {
		z.LogChangeValue("tokens", z.Tokens, o.Tokens)
		z.Tokens = *o.Tokens
	}
	if o.Embedding != nil {
		z.LogChangeValue("embedding", z.Embedding, o.Embedding)
		z.Embedding = *o.Embedding
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
