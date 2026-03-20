package convo

import (
	"fmt"
	"strings"
	"time"

	oid "github.com/cupogo/andvari/models/oid"
)

// GetOwnerID 返回所有者ID
func (m *MemoryBasic) GetOwnerID() oid.OID {
	return m.OwnerID
}

// SetOwnerID 设置所有者ID
func (m *MemoryBasic) SetOwnerID(id any) bool {
	if v := oid.Cast(id); v.Valid() {
		m.OwnerID = v
		return true
	}
	return false
}

// ownerID 可否为空
func (m *MemoryBasic) OwnerEmpty() bool {
	return false
}

// GetSubject returns the document subject (key + category + content)
func (z MemoryBasic) GetSubject() string {
	return fmt.Sprintf("%s %s  %s  %s", z.OwnerID.String(), z.Key, z.Cate, z.Content)
}

func (z Memories) Keys() []string {
	keys := make([]string, len(z))
	for i := range z {
		keys[i] = z[i].Key
	}
	return keys
}

// PrettyText 用于格式显式某个人的记忆清单
func (z Memories) PrettyTextForOwner() string {
	if len(z) == 0 {
		return "*No memory yet*"
	}

	var sb strings.Builder
	sb.Grow(len(z) * 50) // ~= line length

	fmt.Fprintf(&sb, "There are %d pieces of memory related to user ID %s.\n", len(z), z[0].OwnerID)

	noContent := len(z[0].Content) == 0

	if noContent {
		sb.WriteString("you need to use the tool memory_recall again to retrieve the memory content.\n")

		sb.WriteString("\n| updated | cate | key |\n")
		sb.WriteString("| ---- | ---- | ---- |\n")
	} else {
		sb.WriteString("\n| updated | cate | key | content |\n")
		sb.WriteString("| ---- | ---- | ---- | ---- |\n")
	}

	for _, m := range z {
		if noContent {
			fmt.Fprintf(&sb, "| %s | %s | %s |\n",
				m.GetUpdated().Format(time.DateOnly), m.Cate, m.Key)
		} else {
			fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
				m.GetUpdated().Format(time.DateOnly), m.Cate, m.Key, m.Content)
		}

	}

	return sb.String()
}
