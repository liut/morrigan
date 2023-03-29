// This file is generated - Do Not Edit.

package stores

import (
	"context"

	"github.com/liut/morrigan/pkg/models/qas"
)

// type Document = qas.Document
// type DocumentBasic = qas.DocumentBasic
// type DocumentSet = qas.DocumentSet
// type Documents = qas.Documents

func init() {
	RegisterModel((*qas.Document)(nil))
}

type QaStore interface {
	qaStoreX

	ListDocument(ctx context.Context, spec *DocumentSpec) (data qas.Documents, total int, err error)
	GetDocument(ctx context.Context, id string) (obj *qas.Document, err error)
	CreateDocument(ctx context.Context, in qas.DocumentBasic) (obj *qas.Document, err error)
	UpdateDocument(ctx context.Context, id string, in qas.DocumentSet) error
	DeleteDocument(ctx context.Context, id string) error
}

type DocumentSpec struct {
	PageSpec
	ModelSpec

	// 主标题
	Title string `extensions:"x-order=A" form:"title" json:"title"`
	// 小节标题
	Heading string `extensions:"x-order=B" form:"heading" json:"heading"`
	// 内容
	Content string `extensions:"x-order=C" form:"content" json:"content"`
}

func (spec *DocumentSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftMatch(q, "title", spec.Title, false)
	q, _ = siftMatch(q, "heading", spec.Heading, false)
	q, _ = siftMatch(q, "content", spec.Content, false)

	return q
}
func (spec *DocumentSpec) CanSort(k string) bool {
	switch k {
	case "heading":
		return true
	default:
		return spec.ModelSpec.CanSort(k)
	}
}

type qaStore struct {
	w *Wrap
}

func (s *qaStore) ListDocument(ctx context.Context, spec *DocumentSpec) (data qas.Documents, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *qaStore) GetDocument(ctx context.Context, id string) (obj *qas.Document, err error) {
	obj = new(qas.Document)
	err = s.w.db.GetModel(ctx, obj, id)

	return
}
func (s *qaStore) CreateDocument(ctx context.Context, in qas.DocumentBasic) (obj *qas.Document, err error) {
	obj = qas.NewDocumentWithBasic(in)
	err = s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		if err = dbBeforeSaveDocument(ctx, tx, obj); err != nil {
			return err
		}
		dbOpModelMeta(ctx, tx, obj)
		err = dbInsert(ctx, tx, obj)
		return err
	})
	return
}
func (s *qaStore) UpdateDocument(ctx context.Context, id string, in qas.DocumentSet) error {
	exist := new(qas.Document)
	if err := getModelWithPKID(ctx, s.w.db, exist, id); err != nil {
		return err
	}
	exist.SetWith(in)
	return s.w.db.RunInTx(ctx, nil, func(ctx context.Context, tx pgTx) (err error) {
		exist.SetIsUpdate(true)
		if err = dbBeforeSaveDocument(ctx, tx, exist); err != nil {
			return
		}
		dbOpModelMeta(ctx, tx, exist)
		return dbUpdate(ctx, tx, exist)
	})
}
func (s *qaStore) DeleteDocument(ctx context.Context, id string) error {
	obj := new(qas.Document)
	return s.w.db.DeleteModel(ctx, obj, id)
}
