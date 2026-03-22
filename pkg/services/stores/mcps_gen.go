// This file is generated - Do Not Edit.

package stores

import (
	"context"

	"github.com/liut/morign/pkg/models/mcps"
)

// type MCPServer = mcps.Server

func init() {
	RegisterModel((*mcps.Server)(nil))
}

type MCPStore interface {
	ListServer(ctx context.Context, spec *MCPServerSpec) (data mcps.Servers, total int, err error)
	GetServer(ctx context.Context, id string) (obj *mcps.Server, err error)
	CreateServer(ctx context.Context, in mcps.ServerBasic) (obj *mcps.Server, err error)
	UpdateServer(ctx context.Context, id string, in mcps.ServerSet) error
	DeleteServer(ctx context.Context, id string) error
}

type MCPServerSpec struct {
	PageSpec
	ModelSpec

	// 名称
	Name string `extensions:"x-order=A" form:"name" json:"name"`
	// 传输类型 (支持混合解码)
	//  * `stdIO` - 标准IO
	//  * `sse`
	//  * `streamable`
	//  * `inMemory` - 内部运行
	TransType string `extensions:"x-order=B" form:"transType" json:"transType" swaggertype:"string"`
	// 是否激活
	IsActive string `extensions:"x-order=C" form:"isActive" json:"isActive"`
	// 连接状态
	//  * `disconnected` - 断开
	//  * `connecting` - 连接中
	//  * `connected` - 已连接
	Status mcps.Status `extensions:"x-order=D" form:"status" json:"status" swaggertype:"string"`
}

func (spec *MCPServerSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftMatch(q, "name", spec.Name, false)
	if len(spec.TransType) > 0 {
		var v mcps.TransType
		if err := v.Decode(spec.TransType); err == nil {
			q = q.Where("?TableAlias.trans_type = ?", v)
		}
	}
	q, _ = siftEqual(q, "is_active", spec.IsActive, false)
	q, _ = siftEqual(q, "status", spec.Status, false)

	return q
}

type mcpStore struct {
	w *Wrap
}

func (s *mcpStore) ListServer(ctx context.Context, spec *MCPServerSpec) (data mcps.Servers, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	if err == nil {
		err = s.afterListServer(ctx, spec, data)
	}
	return
}
func (s *mcpStore) GetServer(ctx context.Context, id string) (obj *mcps.Server, err error) {
	obj = new(mcps.Server)
	if err = dbGetWith(ctx, s.w.db, obj, "name", "=", id); err != nil && obj.SetID(id) {
		err = dbGetWithPK(ctx, s.w.db, obj)
	}
	if err == nil {
		err = s.afterLoadServer(ctx, obj)
	}
	return
}
func (s *mcpStore) CreateServer(ctx context.Context, in mcps.ServerBasic) (obj *mcps.Server, err error) {
	obj = mcps.NewServerWithBasic(in)
	if obj.Name == "" {
		err = ErrEmptyKey
		return
	}
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj, "name")
	return
}
func (s *mcpStore) UpdateServer(ctx context.Context, id string, in mcps.ServerSet) error {
	exist := new(mcps.Server)
	if err := dbGetWithPKID(ctx, s.w.db, exist, id); err != nil {
		return err
	}
	exist.SetIsUpdate(true)
	exist.SetWith(in)
	dbMetaUp(ctx, s.w.db, exist)
	return dbUpdate(ctx, s.w.db, exist)
}
func (s *mcpStore) DeleteServer(ctx context.Context, id string) error {
	obj := new(mcps.Server)
	return s.w.db.DeleteModel(ctx, obj, id)
}
