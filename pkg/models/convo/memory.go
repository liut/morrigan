package convo

import (
	"fmt"

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
	return fmt.Sprintf("%s  %s  %s", z.Key, z.Cate, z.Content)
}

// GetSubject returns the subject for Memory
func (m *Memory) GetSubject() string {
	return m.MemoryBasic.GetSubject()
}
