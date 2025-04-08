// This file is generated - Do Not Edit.

package stores

import (
	"context"

	"github.com/liut/morrigan/pkg/models/qas"
)

// type ChatLog = qas.ChatLog
// type DocMatch = qas.DocMatch
// type DocMatches = qas.DocMatches
// type QaDocVector = qas.DocVector
// type QaDocument = qas.Document
// type Prompt = qas.Prompt

func init() {
	RegisterModel((*qas.Document)(nil), (*qas.DocVector)(nil), (*qas.Prompt)(nil), (*qas.ChatLog)(nil))
}

type QaStore interface {
	qaStoreX

	ListDocument(ctx context.Context, spec *QaDocumentSpec) (data qas.Documents, total int, err error)
	GetDocument(ctx context.Context, id string) (obj *qas.Document, err error)
	CreateDocument(ctx context.Context, in qas.DocumentBasic) (obj *qas.Document, err error)
	UpdateDocument(ctx context.Context, id string, in qas.DocumentSet) error
	DeleteDocument(ctx context.Context, id string) error

	GetDocVector(ctx context.Context, id string) (obj *qas.DocVector, err error)
	CreateDocVector(ctx context.Context, in qas.DocVectorBasic) (obj *qas.DocVector, err error)
	DeleteDocVector(ctx context.Context, id string) error

	CreatePrompt(ctx context.Context, in qas.PromptBasic) (obj *qas.Prompt, err error)
	UpdatePrompt(ctx context.Context, id string, in qas.PromptSet) error
	DeletePrompt(ctx context.Context, id string) error

	CreateChatLog(ctx context.Context, in qas.ChatLogBasic) (obj *qas.ChatLog, err error)
	GetChatLog(ctx context.Context, id string) (obj *qas.ChatLog, err error)
	ListChatLog(ctx context.Context, spec *ChatLogSpec) (data qas.ChatLogs, total int, err error)
	DeleteChatLog(ctx context.Context, id string) error
}

type QaDocumentSpec struct {
	PageSpec
	ModelSpec

	// 主标题 名称
	Title string `extensions:"x-order=A" form:"title" json:"title"`
	// 小节标题 属性 类别
	Heading string `extensions:"x-order=B" form:"heading" json:"heading"`
	// 内容 值
	Content string `extensions:"x-order=C" form:"content" json:"content"`
}

func (spec *QaDocumentSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftMatch(q, "title", spec.Title, false)
	q, _ = siftMatch(q, "heading", spec.Heading, false)
	q, _ = siftMatch(q, "content", spec.Content, false)

	return q
}
func (spec *QaDocumentSpec) CanSort(k string) bool {
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

type qaStore struct {
	w *Wrap
}

func (s *qaStore) ListDocument(ctx context.Context, spec *QaDocumentSpec) (data qas.Documents, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *qaStore) GetDocument(ctx context.Context, id string) (obj *qas.Document, err error) {
	obj = new(qas.Document)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *qaStore) CreateDocument(ctx context.Context, in qas.DocumentBasic) (obj *qas.Document, err error) {
	err = s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		obj = qas.NewDocumentWithBasic(in)
		if err = dbBeforeSaveQaDocument(ctx, tx, obj); err != nil {
			return
		}
		dbMetaUp(ctx, tx, obj)
		err = dbInsert(ctx, tx, obj)
		return err
	})
	if err == nil {
		err = s.afterCreatedQaDocument(ctx, obj)
	}
	return
}
func (s *qaStore) UpdateDocument(ctx context.Context, id string, in qas.DocumentSet) error {
	return s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		exist := new(qas.Document)
		if err = dbGetWithPKID(ctx, tx, exist, id); err != nil {
			return err
		}
		exist.SetIsUpdate(true)
		exist.SetWith(in)
		if err = dbBeforeSaveQaDocument(ctx, tx, exist); err != nil {
			return err
		}
		dbMetaUp(ctx, tx, exist)
		return dbUpdate(ctx, tx, exist)
	})
}
func (s *qaStore) DeleteDocument(ctx context.Context, id string) error {
	obj := new(qas.Document)
	if err := dbGetWithPKID(ctx, s.w.db, obj, id); err != nil {
		return err
	}
	return s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		err = dbDeleteM(ctx, tx, s.w.db.Schema(), s.w.db.SchemaCrap(), obj)
		if err != nil {
			return
		}
		return dbAfterDeleteQaDocument(ctx, tx, obj)
	})
}

func (s *qaStore) GetDocVector(ctx context.Context, id string) (obj *qas.DocVector, err error) {
	obj = new(qas.DocVector)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *qaStore) CreateDocVector(ctx context.Context, in qas.DocVectorBasic) (obj *qas.DocVector, err error) {
	obj = qas.NewDocVectorWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *qaStore) DeleteDocVector(ctx context.Context, id string) error {
	obj := new(qas.DocVector)
	return s.w.db.DeleteModel(ctx, obj, id)
}

func (s *qaStore) CreatePrompt(ctx context.Context, in qas.PromptBasic) (obj *qas.Prompt, err error) {
	err = s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		obj = qas.NewPromptWithBasic(in)
		if err = dbBeforeSavePrompt(ctx, tx, obj); err != nil {
			return
		}
		if obj.Text == "" {
			err = ErrEmptyKey
			return
		}
		dbMetaUp(ctx, tx, obj)
		err = dbInsert(ctx, tx, obj, "prompt")
		return err
	})
	return
}
func (s *qaStore) UpdatePrompt(ctx context.Context, id string, in qas.PromptSet) error {
	return s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		exist := new(qas.Prompt)
		if err = dbGetWithPKID(ctx, tx, exist, id); err != nil {
			return err
		}
		exist.SetIsUpdate(true)
		exist.SetWith(in)
		if err = dbBeforeSavePrompt(ctx, tx, exist); err != nil {
			return err
		}
		dbMetaUp(ctx, tx, exist)
		return dbUpdate(ctx, tx, exist)
	})
}
func (s *qaStore) DeletePrompt(ctx context.Context, id string) error {
	obj := new(qas.Prompt)
	return s.w.db.DeleteModel(ctx, obj, id)
}

func (s *qaStore) CreateChatLog(ctx context.Context, in qas.ChatLogBasic) (obj *qas.ChatLog, err error) {
	obj = qas.NewChatLogWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *qaStore) GetChatLog(ctx context.Context, id string) (obj *qas.ChatLog, err error) {
	obj = new(qas.ChatLog)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *qaStore) ListChatLog(ctx context.Context, spec *ChatLogSpec) (data qas.ChatLogs, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *qaStore) DeleteChatLog(ctx context.Context, id string) error {
	obj := new(qas.ChatLog)
	return s.w.db.DeleteModel(ctx, obj, id)
}
