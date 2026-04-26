// This file is generated - Do Not Edit.

package capability

import (
	comm "github.com/cupogo/andvari/models/comm"
	oid "github.com/cupogo/andvari/models/oid"
	corpus "github.com/liut/morign/pkg/models/corpus"
)

// consts of Capability API
const (
	CapabilityTable = "api_capability"
	CapabilityAlias = "ac"
	CapabilityLabel = "capability"
	CapabilityTypID = "capability"
)

// Capability API 能力描述
type Capability struct {
	comm.BaseModel `bun:"table:api_capability,alias:ac" json:"-"`

	comm.DefaultModel

	CapabilityBasic

	comm.MetaField
} // @name capability

type CapabilityBasic struct {
	// operationId（可为空，公开接口无此字段）
	OperationID string `bson:"operationID" bun:",notnull,type:varchar(63)" extensions:"x-order=A" form:"operationID" json:"operationID" pg:",notnull,type:varchar(63)"`
	// API 路径，如 /api/accounts/{id}
	Endpoint string `bson:"endpoint" bun:",notnull,type:varchar(125)" extensions:"x-order=B" form:"endpoint" json:"endpoint" pg:",notnull,type:varchar(125)"`
	// HTTP 方法 GET/POST/PUT/DELETE 等
	Method string `bson:"method" bun:",notnull,type:varchar(10)" extensions:"x-order=C" form:"method" json:"method" pg:",notnull,type:varchar(10)"`
	// 简短描述
	Summary string `bson:"summary" bun:",notnull,type:text" extensions:"x-order=D" form:"summary" json:"summary" pg:",notnull,type:text"`
	// 详细描述
	Description string `bson:"description" bun:",notnull,type:text" extensions:"x-order=E" form:"description" json:"description" pg:",notnull,type:text"`
	// 参数结构 JSON
	Parameters []SwaggerParam `bson:"parameters" bun:",notnull,type:jsonb,default:'[]'" extensions:"x-order=F" json:"parameters,omitempty" pg:",notnull,type:jsonb,default:'[]'"`
	// 响应结构 map[code]SwaggerResponse
	Responses map[string]SwaggerResponse `bson:"responses" bun:",notnull,type:jsonb,default:'{}'" extensions:"x-order=G" json:"responses,omitempty" pg:",notnull,type:jsonb,default:'{}'"`
	// API 标签
	Tags []string `bson:"tags" bun:",notnull,type:jsonb,default:'[]'" extensions:"x-order=H" json:"tags,omitempty" pg:",notnull,type:jsonb,default:'[]'"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name capabilityBasic

type Capabilities []Capability

// Creating function call to it's inner fields defined hooks
func (z *Capability) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtFile))
	}

	return z.DefaultModel.Creating()
}
func NewCapabilityWithBasic(in CapabilityBasic) *Capability {
	obj := &Capability{
		CapabilityBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewCapabilityWithID(id any) *Capability {
	obj := new(Capability)
	_ = obj.SetID(id)
	return obj
}
func (_ *Capability) IdentityLabel() string { return CapabilityLabel }
func (_ *Capability) IdentityModel() string { return CapabilityTypID }
func (_ *Capability) IdentityTable() string { return CapabilityTable }
func (_ *Capability) IdentityAlias() string { return CapabilityAlias }

type CapabilitySet struct {
	// operationId（可为空，公开接口无此字段）
	OperationID *string `extensions:"x-order=A" json:"operationID"`
	// API 路径，如 /api/accounts/{id}
	Endpoint *string `extensions:"x-order=B" json:"endpoint"`
	// HTTP 方法 GET/POST/PUT/DELETE 等
	Method *string `extensions:"x-order=C" json:"method"`
	// 简短描述
	Summary *string `extensions:"x-order=D" json:"summary"`
	// 详细描述
	Description *string `extensions:"x-order=E" json:"description"`
	// 参数结构 JSON
	Parameters *[]SwaggerParam `extensions:"x-order=F" json:"parameters,omitempty"`
	// 响应结构 map[code]SwaggerResponse
	Responses *map[string]SwaggerResponse `extensions:"x-order=G" json:"responses,omitempty"`
	// API 标签
	Tags *[]string `extensions:"x-order=H" json:"tags,omitempty"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name capabilitySet

func (z *Capability) SetWith(o CapabilitySet) {
	if o.OperationID != nil && z.OperationID != *o.OperationID {
		z.LogChangeValue("operation_id", z.OperationID, o.OperationID)
		z.OperationID = *o.OperationID
	}
	if o.Endpoint != nil && z.Endpoint != *o.Endpoint {
		z.LogChangeValue("endpoint", z.Endpoint, o.Endpoint)
		z.Endpoint = *o.Endpoint
	}
	if o.Method != nil && z.Method != *o.Method {
		z.LogChangeValue("method", z.Method, o.Method)
		z.Method = *o.Method
	}
	if o.Summary != nil && z.Summary != *o.Summary {
		z.LogChangeValue("summary", z.Summary, o.Summary)
		z.Summary = *o.Summary
	}
	if o.Description != nil && z.Description != *o.Description {
		z.LogChangeValue("description", z.Description, o.Description)
		z.Description = *o.Description
	}
	if o.Parameters != nil {
		z.LogChangeValue("parameters", z.Parameters, o.Parameters)
		z.Parameters = *o.Parameters
	}
	if o.Responses != nil {
		z.LogChangeValue("responses", z.Responses, o.Responses)
		z.Responses = *o.Responses
	}
	if o.Tags != nil {
		z.LogChangeValue("tags", z.Tags, o.Tags)
		z.Tags = *o.Tags
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *CapabilityBasic) MetaAddKVs(args ...any) *CapabilityBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *CapabilitySet) MetaAddKVs(args ...any) *CapabilitySet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// consts of CapabilityVector API
const (
	CapabilityVectorTable = "api_capability_vector"
	CapabilityVectorAlias = "cv"
	CapabilityVectorLabel = "capabilityVector"
	CapabilityVectorTypID = "capabilityCapabilityVector"
)

// CapabilityVector API 能力向量
type CapabilityVector struct {
	comm.BaseModel `bun:"table:api_capability_vector,alias:cv" json:"-"`

	comm.DefaultModel

	CapabilityVectorBasic

	comm.MetaField
} // @name capabilityCapabilityVector

type CapabilityVectorBasic struct {
	// 关联的 Capability ID
	CapID oid.OID `bson:"capID" bun:"cap_id,notnull" extensions:"x-order=A" json:"capID" pg:"cap_id,notnull" swaggertype:"string"`
	// 主题 基于 summary + description 等生成
	Subject string `bson:"subject" bun:"subject,notnull,type:text" extensions:"x-order=B" form:"subject" json:"subject" pg:"subject,notnull,type:text"`
	// 语义向量 1024 维
	Vector corpus.Vector `bson:"vector" bun:"embedding,notnull,type:vector(1024)" extensions:"x-order=C" json:"vector" pg:"embedding,notnull,type:vector(1024)"`
	// for meta update
	MetaDiff *comm.MetaDiff `bson:"-" bun:"-" json:"metaUp,omitempty" pg:"-" swaggerignore:"true"`
} // @name capabilityCapabilityVectorBasic

type CapabilityVectors []CapabilityVector

// Creating function call to it's inner fields defined hooks
func (z *CapabilityVector) Creating() error {
	if z.IsZeroID() {
		z.SetID(oid.NewID(oid.OtEvent))
	}

	return z.DefaultModel.Creating()
}
func NewCapabilityVectorWithBasic(in CapabilityVectorBasic) *CapabilityVector {
	obj := &CapabilityVector{
		CapabilityVectorBasic: in,
	}
	_ = obj.MetaUp(in.MetaDiff)
	return obj
}
func NewCapabilityVectorWithID(id any) *CapabilityVector {
	obj := new(CapabilityVector)
	_ = obj.SetID(id)
	return obj
}
func (_ *CapabilityVector) IdentityLabel() string { return CapabilityVectorLabel }
func (_ *CapabilityVector) IdentityModel() string { return CapabilityVectorTypID }
func (_ *CapabilityVector) IdentityTable() string { return CapabilityVectorTable }
func (_ *CapabilityVector) IdentityAlias() string { return CapabilityVectorAlias }

type CapabilityVectorSet struct {
	// 关联的 Capability ID
	CapID *string `extensions:"x-order=A" json:"capID"`
	// 主题 基于 summary + description 等生成
	Subject *string `extensions:"x-order=B" json:"subject"`
	// 语义向量 1024 维
	Vector *corpus.Vector `extensions:"x-order=C" json:"vector"`
	// for meta update
	MetaDiff *comm.MetaDiff `json:"metaUp,omitempty" swaggerignore:"true"`
} // @name capabilityCapabilityVectorSet

func (z *CapabilityVector) SetWith(o CapabilityVectorSet) {
	if o.CapID != nil {
		if id := oid.Cast(*o.CapID); z.CapID != id {
			z.LogChangeValue("cap_id", z.CapID, id)
			z.CapID = id
		}
	}
	if o.Subject != nil && z.Subject != *o.Subject {
		z.LogChangeValue("subject", z.Subject, o.Subject)
		z.Subject = *o.Subject
	}
	if o.Vector != nil {
		z.LogChangeValue("embedding", z.Vector, o.Vector)
		z.Vector = *o.Vector
	}
	if o.MetaDiff != nil && z.MetaUp(o.MetaDiff) {
		z.SetChange("meta")
	}
}
func (in *CapabilityVectorBasic) MetaAddKVs(args ...any) *CapabilityVectorBasic {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}
func (in *CapabilityVectorSet) MetaAddKVs(args ...any) *CapabilityVectorSet {
	in.MetaDiff = comm.MetaDiffAddKVs(in.MetaDiff, args...)
	return in
}

// consts of SwaggerParam swagger
const (
	SwaggerParamLabel = "swaggerParam"
	SwaggerParamTypID = "capabilitySwaggerParam"
)

// SwaggerParam swagger 参数定义
type SwaggerParam struct {
	// 参数类型
	Type string `extensions:"x-order=A" form:"type" json:"type" yaml:"type"`
	// 参数描述
	Description string `extensions:"x-order=B" form:"description" json:"description" yaml:"description"`
	// 参数名称
	Name string `extensions:"x-order=C" form:"name" json:"name" yaml:"name"`
	// 参数位置 header/query/body/path
	In string `extensions:"x-order=D" form:"in" json:"in" yaml:"in"`
	// 是否必填
	Required bool `extensions:"x-order=E" form:"required" json:"required,omitempty" yaml:"required"`
	// 参数示例
	Example string `extensions:"x-order=F" form:"example" json:"example,omitempty" yaml:"example"`
	// schema 定义
	Schema any `extensions:"x-order=G" json:"schema,omitempty" yaml:"schema"`
} // @name capabilitySwaggerParam

// consts of SwaggerSchema swagger
const (
	SwaggerSchemaLabel = "swaggerSchema"
	SwaggerSchemaTypID = "capabilitySwaggerSchema"
)

// SwaggerSchema swagger schema 定义
type SwaggerSchema struct {
	// 引用路径如
	Ref string `extensions:"x-order=A" form:"$ref" json:"$ref,omitempty" yaml:"$ref"`
	// 类型
	Type string `extensions:"x-order=B" form:"type" json:"type,omitempty" yaml:"type"`
	// 描述
	Description string `extensions:"x-order=C" form:"description" json:"description,omitempty" yaml:"description"`
	// allOf 组合
	AllOf []SwaggerSchema `extensions:"x-order=D" json:"allOf,omitempty" yaml:"allOf"`
	// 属性定义
	Properties map[string]SwaggerSchema `extensions:"x-order=E" json:"properties,omitempty" yaml:"properties"`
} // @name capabilitySwaggerSchema

// consts of SwaggerResponse swagger
const (
	SwaggerResponseLabel = "swaggerResponse"
	SwaggerResponseTypID = "capabilitySwaggerResponse"
)

// SwaggerResponse swagger 响应定义
type SwaggerResponse struct {
	// 响应描述
	Description string `extensions:"x-order=A" form:"description" json:"description" yaml:"description"`
	// schema 定义
	Schema SwaggerSchema `extensions:"x-order=B" json:"schema,omitempty" yaml:"schema"`
} // @name capabilitySwaggerResponse
