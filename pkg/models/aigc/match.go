package aigc

import (
	"github.com/cupogo/andvari/models/oid"
)

// MatchResult 通用匹配结果结构
// 用于向量检索返回的匹配结果
type MatchResult struct {
	// DocID 关联文档的 ID
	DocID oid.OID `bson:"docID" extensions:"x-order=A" json:"docID" swaggertype:"string"`
	// 主题
	Subject string `bson:"subject" extensions:"x-order=B" form:"subject" json:"subject"`
	// 相似度
	Similarity float32 `bson:"similarity" extensions:"x-order=C" json:"similarity"`
}

// MatchResults is a slice of MatchResult
type MatchResults []MatchResult

// DocIDs returns the document IDs from all match results
func (z MatchResults) DocIDs() (out oid.OIDs) {
	out = make(oid.OIDs, 0, len(z))
	for _, m := range z {
		out = append(out, m.DocID)
	}
	return
}
