package cob

import (
	"fmt"
	"strings"

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
	if len(z) == 0 {
		return "No relevant information found in the knowledge base."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant documents in the knowledge base:\n\n", len(z)))
	for _, doc := range z {
		sb.WriteString("---\nID: ")
		sb.WriteString(doc.StringID())
		sb.WriteString("\n\n## ")
		sb.WriteString(doc.Title)
		sb.WriteString("\n\n### ")
		sb.WriteString(doc.Heading)
		sb.WriteString("\n\n")
		sb.WriteString(doc.Content)
		sb.WriteString("\n\n")
	}
	return sb.String()
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
