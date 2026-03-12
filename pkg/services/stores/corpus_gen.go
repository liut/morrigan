// This file is generated - Do Not Edit.

package stores

import (
	"context"

	"github.com/liut/morign/pkg/models/corpus"
)

// type ChatLog = corpus.ChatLog
// type DocMatch = corpus.DocMatch
// type DocMatches = corpus.DocMatches
// type CobDocVector = corpus.DocVector
// type CobDocument = corpus.Document

func init() {
	RegisterModel((*corpus.Document)(nil), (*corpus.DocVector)(nil), (*corpus.ChatLog)(nil))
}

type CobStore interface {
	CobStoreX

	ListDocument(ctx context.Context, spec *CobDocumentSpec) (data corpus.Documents, total int, err error)
	GetDocument(ctx context.Context, id string) (obj *corpus.Document, err error)
	CreateDocument(ctx context.Context, in corpus.DocumentBasic) (obj *corpus.Document, err error)
	UpdateDocument(ctx context.Context, id string, in corpus.DocumentSet) error
	DeleteDocument(ctx context.Context, id string) error

	GetDocVector(ctx context.Context, id string) (obj *corpus.DocVector, err error)
	CreateDocVector(ctx context.Context, in corpus.DocVectorBasic) (obj *corpus.DocVector, err error)
	DeleteDocVector(ctx context.Context, id string) error

	CreateChatLog(ctx context.Context, in corpus.ChatLogBasic) (obj *corpus.ChatLog, err error)
	GetChatLog(ctx context.Context, id string) (obj *corpus.ChatLog, err error)
	ListChatLog(ctx context.Context, spec *ChatLogSpec) (data corpus.ChatLogs, total int, err error)
	DeleteChatLog(ctx context.Context, id string) error
}

type CobDocumentSpec struct {
	PageSpec
	ModelSpec

	// 主标题 名称
	Title string `extensions:"x-order=A" form:"title" json:"title"`
	// 小节标题 属性 类别
	Heading string `extensions:"x-order=B" form:"heading" json:"heading"`
	// 内容 值
	Content string `extensions:"x-order=C" form:"content" json:"content"`
}

func (spec *CobDocumentSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftMatch(q, "title", spec.Title, false)
	q, _ = siftMatch(q, "heading", spec.Heading, false)
	q, _ = siftMatch(q, "content", spec.Content, false)

	return q
}
func (spec *CobDocumentSpec) CanSort(k string) bool {
	switch k {
	case "heading":
		return true
	default:
		return spec.ModelSpec.CanSort(k)
	}
}

type ChatLogSpec struct {
	PageSpec
	ModelSpec

	// 会话ID
	ChatID string `extensions:"x-order=A" form:"csid" json:"csid"`
}

func (spec *ChatLogSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftOID(q, "csid", spec.ChatID, false)

	return q
}

type corpuStore struct {
	w *Wrap
}

func (s *corpuStore) ListDocument(ctx context.Context, spec *CobDocumentSpec) (data corpus.Documents, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *corpuStore) GetDocument(ctx context.Context, id string) (obj *corpus.Document, err error) {
	obj = new(corpus.Document)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *corpuStore) CreateDocument(ctx context.Context, in corpus.DocumentBasic) (obj *corpus.Document, err error) {
	obj = corpus.NewDocumentWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	if err == nil {
		err = s.afterCreatedCobDocument(ctx, obj)
	}
	return
}
func (s *corpuStore) UpdateDocument(ctx context.Context, id string, in corpus.DocumentSet) error {
	exist := new(corpus.Document)
	if err := dbGetWithPKID(ctx, s.w.db, exist, id); err != nil {
		return err
	}
	exist.SetIsUpdate(true)
	exist.SetWith(in)
	dbMetaUp(ctx, s.w.db, exist)
	return dbUpdate(ctx, s.w.db, exist)
}
func (s *corpuStore) DeleteDocument(ctx context.Context, id string) error {
	obj := new(corpus.Document)
	if err := dbGetWithPKID(ctx, s.w.db, obj, id); err != nil {
		return err
	}
	return s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		err = dbDeleteM(ctx, tx, s.w.db.Schema(), s.w.db.SchemaCrap(), obj)
		if err != nil {
			return
		}
		return dbAfterDeleteCobDocument(ctx, tx, obj)
	})
}

func (s *corpuStore) GetDocVector(ctx context.Context, id string) (obj *corpus.DocVector, err error) {
	obj = new(corpus.DocVector)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *corpuStore) CreateDocVector(ctx context.Context, in corpus.DocVectorBasic) (obj *corpus.DocVector, err error) {
	obj = corpus.NewDocVectorWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *corpuStore) DeleteDocVector(ctx context.Context, id string) error {
	obj := new(corpus.DocVector)
	return s.w.db.DeleteModel(ctx, obj, id)
}

func (s *corpuStore) CreateChatLog(ctx context.Context, in corpus.ChatLogBasic) (obj *corpus.ChatLog, err error) {
	obj = corpus.NewChatLogWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *corpuStore) GetChatLog(ctx context.Context, id string) (obj *corpus.ChatLog, err error) {
	obj = new(corpus.ChatLog)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *corpuStore) ListChatLog(ctx context.Context, spec *ChatLogSpec) (data corpus.ChatLogs, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *corpuStore) DeleteChatLog(ctx context.Context, id string) error {
	obj := new(corpus.ChatLog)
	return s.w.db.DeleteModel(ctx, obj, id)
}
