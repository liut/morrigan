package qas

import (
	"github.com/cupogo/andvari/models/oid"
	"github.com/cupogo/andvari/utils/array"
)

func (z Prompts) DocumentIDs() (out oid.OIDs) {
	m := make(map[oid.OID]array.Empty)
	for _, p := range z {
		m[p.DocID] = array.Empty{}
	}
	for k := range m {
		out = append(out, k)
	}
	return
}
