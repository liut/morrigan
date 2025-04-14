package qas

import (
	"bytes"
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

func (z Documents) MarkdownText() string {
	var buf bytes.Buffer
	for _, doc := range z {
		buf.WriteString("---")
		buf.WriteString("ID: " + doc.StringID())
		buf.WriteString("\n\n")
		buf.WriteString("## " + doc.Title)
		buf.WriteString("\n\n")
		buf.WriteString("### " + doc.Heading)
		buf.WriteString("\n\n")
		buf.WriteString(doc.Content)
		buf.WriteString("\n\n")
	}
	return buf.String()
}

func (z Documents) Headings() []string {
	headings := make([]string, len(z))
	for i, doc := range z {
		headings[i] = doc.Heading
	}
	return headings
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

func (z DocMatches) Subjects() (out []string) {
	out = make([]string, len(z))
	for i := range z {
		out[i] = z[i].Subject
	}
	return
}
