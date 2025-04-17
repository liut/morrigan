// This file is generated - Do Not Edit.

package stores

import (
	"context"

	"github.com/liut/morrigan/pkg/models/mcps"
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
	//  * `http`
	//  * `inMemory` - 内部运行
	TransType string `extensions:"x-order=B" form:"transType" json:"transType" swaggertype:"string"`
	// 状态
	//  * `stopped` - 已停止
	//  * `running` - 运行中
	Status mcps.Status `extensions:"x-order=C" form:"status" json:"status" swaggertype:"string"`
}

func (spec *MCPServerSpec) Sift(q *ormQuery) *ormQuery {
	q = spec.ModelSpec.Sift(q)
	q, _ = siftMatch(q, "name", spec.Name, false)
	if len(spec.TransType) > 0 {
		var v mcps.TransType
		if err := v.Decode(spec.TransType); err == nil {
			q = q.Where("trans_type = ?", v)
		}
	}
	q, _ = siftEqual(q, "status", spec.Status, false)

	return q
}

type mcpStore struct {
	w *Wrap
}

func (s *mcpStore) ListServer(ctx context.Context, spec *MCPServerSpec) (data mcps.Servers, total int, err error) {
	total, err = s.w.db.ListModel(ctx, spec, &data)
	return
}
func (s *mcpStore) GetServer(ctx context.Context, id string) (obj *mcps.Server, err error) {
	obj = new(mcps.Server)
	err = dbGetWithPKID(ctx, s.w.db, obj, id)

	return
}
func (s *mcpStore) CreateServer(ctx context.Context, in mcps.ServerBasic) (obj *mcps.Server, err error) {
	obj = mcps.NewServerWithBasic(in)
	dbMetaUp(ctx, s.w.db, obj)
	err = dbInsert(ctx, s.w.db, obj)
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
