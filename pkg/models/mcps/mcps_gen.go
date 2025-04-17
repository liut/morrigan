// This file is generated - Do Not Edit.

package mcps

import (
	"fmt"

	comm "github.com/cupogo/andvari/models/comm"
	oid "github.com/cupogo/andvari/models/oid"
)

// MCP 传输类型
type TransType int8

const (
	TransTypeStdIO    TransType = 1 + iota //  1 标准IO
	TransTypeSSE                           //  2 SSE
	TransTypeHTTP                          //  3 HTTP
	TransTypeInMemory                      //  4 内部运行
)

func (z *TransType) Decode(s string) error {
	switch s {
	case "1", "stdIO", "StdIO":
		*z = TransTypeStdIO
	case "2", "sse", "SSE":
		*z = TransTypeSSE
	case "3", "http", "HTTP":
		*z = TransTypeHTTP
	case "4", "inMemory", "InMemory":
		*z = TransTypeInMemory
	default:
		return fmt.Errorf("invalid transType: %q", s)
	}
	return nil
}
func (z *TransType) UnmarshalText(b []byte) error {
	return z.Decode(string(b))
}
func (z TransType) String() string {
	switch z {
	case TransTypeStdIO:
		return "stdIO"
	case TransTypeSSE:
		return "sse"
	case TransTypeHTTP:
		return "http"
	case TransTypeInMemory:
		return "inMemory"
	default:
		return fmt.Sprintf("transType %d", int8(z))
	}
}
func (z TransType) MarshalText() ([]byte, error) {
	return []byte(z.String()), nil
}

// 状态
type Status int8

const (
	StatusStopped Status = 0 + iota //  0 已停止
	StatusRunning                   //  1 运行中
)

func (z *Status) Decode(s string) error {
	switch s {
	case "0", "stopped", "Stopped":
		*z = StatusStopped
	case "1", "running", "Running":
		*z = StatusRunning
	default:
		return fmt.Errorf("invalid status: %q", s)
	}
	return nil
}
func (z *Status) UnmarshalText(b []byte) error {
	return z.Decode(string(b))
}
func (z Status) String() string {
	switch z {
	case StatusStopped:
		return "stopped"
	case StatusRunning:
		return "running"
	default:
		return fmt.Sprintf("status %d", int8(z))
	}
}
func (z Status) MarshalText() ([]byte, error) {
	return []byte(z.String()), nil
}

// consts of Server 服务器
const (
	ServerTable = "qa_mcp_server"
	ServerAlias = "s"
	ServerLabel = "server"
	ServerTypID = "mcpsServer"
)

// Server 服务器
type Server struct {
	comm.BaseModel `bun:"table:qa_mcp_server,alias:s" json:"-"`

	comm.DefaultModel

	ServerBasic

	// 完整网址 仅对 TransType 为 SSE 或 HTTP 时有效
	URL string `bson:"url" bun:",notnull" extensions:"x-order=D" form:"url" json:"url" pg:",notnull"`

	comm.MetaField
} // @name mcpsServer

type ServerBasic struct {
	// 名称
	Name string `binding:"required" bson:"name" bun:",notnull" extensions:"x-order=A" form:"name" json:"name" pg:",notnull"`
	// 传输类型
	//  * `stdIO` - 标准IO
	//  * `sse`
	//  * `http`
	//  * `inMemory` - 内部运行
	TransType TransType `bson:"transType" bun:",notnull,type:smallint" enums:"stdIO,sse,http,inMemory" extensions:"x-order=B" form:"transType" json:"transType" pg:",notnull,type:smallint" swaggertype:"string"`
	// 指令 仅对 TransType 为 StdIO 时有效
	Command string `bson:"command" bun:",notnull" extensions:"x-order=C" form:"command" json:"command" pg:",notnull"`
	// 状态
	//  * `stopped` - 已停止
	//  * `running` - 运行中
	Status Status `bson:"status" bun:",notnull,type:smallint" enums:"stopped,running" extensions:"x-order=E" form:"status" json:"status" pg:",notnull,type:smallint" swaggertype:"string"`
	// 备注
	Remark string `bson:"remark" bun:",notnull" extensions:"x-order=F" form:"remark" json:"remark" pg:",notnull"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name mcpsServerBasic

type Servers []Server

// Creating function call to it's inner fields defined hooks
func (z *Server) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtFile))
	}

	return z.DefaultModel.Creating()
}
func NewServerWithBasic(in ServerBasic) *Server {
	obj := &Server{
		ServerBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewServerWithID(id any) *Server {
	obj := new(Server)
	_ = obj.SetID(id)
	return obj
}
func (_ *Server) IdentityLabel() string { return ServerLabel }
func (_ *Server) IdentityModel() string { return ServerTypID }
func (_ *Server) IdentityTable() string { return ServerTable }
func (_ *Server) IdentityAlias() string { return ServerAlias }

type ServerSet struct {
	// 名称
	Name *string `extensions:"x-order=A" json:"name"`
	// 传输类型
	//  * `stdIO` - 标准IO
	//  * `sse`
	//  * `http`
	//  * `inMemory` - 内部运行
	TransType *TransType `enums:"stdIO,sse,http,inMemory" extensions:"x-order=B" json:"transType" swaggertype:"string"`
	// 指令 仅对 TransType 为 StdIO 时有效
	Command *string `extensions:"x-order=C" json:"command"`
	// 状态
	//  * `stopped` - 已停止
	//  * `running` - 运行中
	Status *Status `enums:"stopped,running" extensions:"x-order=D" json:"status" swaggertype:"string"`
	// 备注
	Remark *string `extensions:"x-order=E" json:"remark"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name mcpsServerSet

func (z *Server) SetWith(o ServerSet) {
	if o.Name != nil && z.Name != *o.Name {
		z.LogChangeValue("name", z.Name, o.Name)
		z.Name = *o.Name
	}
	if o.TransType != nil && z.TransType != *o.TransType {
		z.LogChangeValue("trans_type", z.TransType, o.TransType)
		z.TransType = *o.TransType
	}
	if o.Command != nil && z.Command != *o.Command {
		z.LogChangeValue("command", z.Command, o.Command)
		z.Command = *o.Command
	}
	if o.Status != nil && z.Status != *o.Status {
		z.LogChangeValue("status", z.Status, o.Status)
		z.Status = *o.Status
	}
	if o.Remark != nil && z.Remark != *o.Remark {
		z.LogChangeValue("remark", z.Remark, o.Remark)
		z.Remark = *o.Remark
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *ServerBasic) MetaAddKVs(args ...any) *ServerBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *ServerSet) MetaAddKVs(args ...any) *ServerSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
