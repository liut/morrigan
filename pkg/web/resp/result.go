package resp

// ViewPatcher 视图修补接口
type ViewPatcher interface {
	PatchView()
}

// ResultData 特定数据集(带JSON数组和总数)，一般用在分页查询结果
type ResultData struct {
	Data  any `json:"data,omitempty"`  // 数据集数组
	Total int `json:"total,omitempty"` // 符合条件的总记录数
}

func (dr *ResultData) PatchView() {
	if v, ok := dr.Data.(ViewPatcher); ok {
		v.PatchView()
	}
}

type ResultID struct {
	ID any `json:"id"` // 主键值，多数时候是字串
}

type ResultOk struct {
	Ok bool `json:"ok"`
}

// RespDone 操作成功返回的结构 (兼容 chi 框架)
type RespDone struct {
	Status int         `json:"status"`
	Data   any         `json:"data,omitempty"`
	Count  int         `json:"count,omitempty"`
	Extra  interface{} `json:"extra,omitempty"`
}
