// This file is generated - Do Not Edit.

package stores

import (
	"context"

	"github.com/liut/morrigan/pkg/models/cob"
)

// type ChatLog = cob.ChatLog
// type DocMatch = cob.DocMatch
// type DocMatches = cob.DocMatches
// type CobDocVector = cob.DocVector
// type CobDocument = cob.Document

func init() {
	RegisterModel((*cob.Document)(nil), (*cob.DocVector)(nil), (*cob.ChatLog)(nil))
}

type CobStore interface {
	CobStoreX

	ListDocument(ctx context.Context, spec *CobDocumentSpec) (data cob.Documents, total int, err error)
	GetDocument(ctx context.Context, id string) (obj *cob.Document, err error)
	CreateDocument(ctx context.Context, in cob.DocumentBasic) (obj *cob.Document, err error)
	UpdateDocument(ctx context.Context, id string, in cob.DocumentSet) error
	DeleteDocument(ctx context.Context, id string) error

	GetDocVector(ctx context.Context, id string) (obj *cob.DocVector, err error)
	CreateDocVector(ctx context.Context, in cob.DocVectorBasic) (obj *cob.DocVector, err error)
	DeleteDocVector(ctx context.Context, id string) error

	CreateChatLog(ctx context.Context, in cob.ChatLogBasic) (obj *cob.ChatLog, err error)
	GetChatLog(ctx context.Context, id string) (obj *cob.ChatLog, err error)
	ListChatLog(ctx context.Context, spec *ChatLogSpec) (data cob.ChatLogs, total int, err error)
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

type cobStore struct {
	w *Wrap
}

func (s *cobStore) ListDocument(ctx context.Context, spec *CobDocumentSpec) (data cob.Documents, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *cobStore) GetDocument(ctx context.Context, id string) (obj *cob.Document, err error) {
	obj = new(cob.Document)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *cobStore) CreateDocument(ctx context.Context, in cob.DocumentBasic) (obj *cob.Document, err error) {
	obj = cob.NewDocumentWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	if err == nil {
		err = s.afterCreatedCobDocument(ctx, obj)
	}
	return
}
func (s *cobStore) UpdateDocument(ctx context.Context, id string, in cob.DocumentSet) error {
	exist := new(cob.Document)
	if err := dbGetWithPKID(ctx, s.w.db, exist, id); err != nil {
		return err
	}
	exist.SetIsUpdate(true)
	exist.SetWith(in)
	dbMetaUp(ctx, s.w.db, exist)
	return dbUpdate(ctx, s.w.db, exist)
}
func (s *cobStore) DeleteDocument(ctx context.Context, id string) error {
	obj := new(cob.Document)
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

func (s *cobStore) GetDocVector(ctx context.Context, id string) (obj *cob.DocVector, err error) {
	obj = new(cob.DocVector)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *cobStore) CreateDocVector(ctx context.Context, in cob.DocVectorBasic) (obj *cob.DocVector, err error) {
	obj = cob.NewDocVectorWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *cobStore) DeleteDocVector(ctx context.Context, id string) error {
	obj := new(cob.DocVector)
	return s.w.db.DeleteModel(ctx, obj, id)
}

func (s *cobStore) CreateChatLog(ctx context.Context, in cob.ChatLogBasic) (obj *cob.ChatLog, err error) {
	obj = cob.NewChatLogWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *cobStore) GetChatLog(ctx context.Context, id string) (obj *cob.ChatLog, err error) {
	obj = new(cob.ChatLog)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *cobStore) ListChatLog(ctx context.Context, spec *ChatLogSpec) (data cob.ChatLogs, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *cobStore) DeleteChatLog(ctx context.Context, id string) error {
	obj := new(cob.ChatLog)
	return s.w.db.DeleteModel(ctx, obj, id)
}
