package qas

import (
	"fmt"

	"github.com/cupogo/andvari/models/oid"
	"github.com/cupogo/andvari/utils/array"
)

func (z *Document) GetSubject() string {
	return fmt.Sprintf("%s %s", z.Title, z.Heading)
}

func (z Documents) IDs() (out oid.OIDs) {
	for _, doc := range z {
		out = append(out, doc.ID)
	}
	return
}

func (z DocMatches) DocumentIDs() (out oid.OIDs) {
	m := make(map[oid.OID]array.Empty)
	for _, p := range z {
		m[p.DocID] = array.Empty{}
	}
	for k := range m {
		out = append(out, k)
	}
	return
}
