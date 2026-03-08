// This file is generated - Do Not Edit.

package stores

import (
	"context"

	"github.com/liut/morign/pkg/models/convo"
)

// type ConvoMessage = convo.Message
// type ConvoSession = convo.Session

func init() {
	RegisterModel((*convo.Session)(nil), (*convo.Message)(nil))
}

type ConvoStore interface {
	ListSession(ctx context.Context, spec *ConvoSessionSpec) (data convo.Sessions, total int, err error)
	GetSession(ctx context.Context, id string) (obj *convo.Session, err error)
	CreateSession(ctx context.Context, in convo.SessionBasic) (obj *convo.Session, err error)
	UpdateSession(ctx context.Context, id string, in convo.SessionSet) error
	DeleteSession(ctx context.Context, id string) error

	ListMessage(ctx context.Context, spec *ConvoMessageSpec) (data convo.Messages, total int, err error)
	GetMessage(ctx context.Context, id string) (obj *convo.Message, err error)
	CreateMessage(ctx context.Context, in convo.MessageBasic) (obj *convo.Message, err error)
	DeleteMessage(ctx context.Context, id string) error
}

type ConvoSessionSpec struct {
	PageSpec
	ModelSpec

	// 标题
	Title string `extensions:"x-order=A" form:"title" json:"title"`
	// 状态
	//  * `open` - 开启
	//  * `closed` - 关闭
	Status convo.SessionStatus `extensions:"x-order=B" form:"status" json:"status" swaggertype:"string"`
	// 所有者编号 (多值使用逗号分隔)
	OwnerID string `extensions:"x-order=C" form:"owner" json:"owner,omitempty"`
}

func (spec *ConvoSessionSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftMatch(q, "title", spec.Title, false)
	q, _ = siftEqual(q, "status", spec.Status, false)
	q, _ = siftOIDs(q, "owner_id", spec.OwnerID, false)

	return q
}

type ConvoMessageSpec struct {
	PageSpec
	ModelSpec

	// 会话编号
	SessionID string `extensions:"x-order=A" form:"session" json:"session"`
	// 角色
	Role string `extensions:"x-order=B" form:"role" json:"role"`
	// TokenCount
	TokenCount int `extensions:"x-order=C" form:"tokenCount" json:"tokenCount"`
}

func (spec *ConvoMessageSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftOID(q, "session_id", spec.SessionID, false)
	q, _ = siftMatch(q, "role", spec.Role, false)
	q, _ = siftEqual(q, "token_count", spec.TokenCount, false)

	return q
}

type convoStore struct {
	w *Wrap
}

func (s *convoStore) ListSession(ctx context.Context, spec *ConvoSessionSpec) (data convo.Sessions, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *convoStore) GetSession(ctx context.Context, id string) (obj *convo.Session, err error) {
	obj = new(convo.Session)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *convoStore) CreateSession(ctx context.Context, in convo.SessionBasic) (obj *convo.Session, err error) {
	obj = convo.NewSessionWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *convoStore) UpdateSession(ctx context.Context, id string, in convo.SessionSet) error {
	exist := new(convo.Session)
	if err := dbGetWithPKID(ctx, s.w.db, exist, id); err != nil {
		return err
	}
	exist.SetIsUpdate(true)
	exist.SetWith(in)
	dbMetaUp(ctx, s.w.db, exist)
	return dbUpdate(ctx, s.w.db, exist)
}
func (s *convoStore) DeleteSession(ctx context.Context, id string) error {
	obj := new(convo.Session)
	return s.w.db.DeleteModel(ctx, obj, id)
}

func (s *convoStore) ListMessage(ctx context.Context, spec *ConvoMessageSpec) (data convo.Messages, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *convoStore) GetMessage(ctx context.Context, id string) (obj *convo.Message, err error) {
	obj = new(convo.Message)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *convoStore) CreateMessage(ctx context.Context, in convo.MessageBasic) (obj *convo.Message, err error) {
	obj = convo.NewMessageWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
	return
}
func (s *convoStore) DeleteMessage(ctx context.Context, id string) error {
	obj := new(convo.Message)
	return s.w.db.DeleteModel(ctx, obj, id)
}
