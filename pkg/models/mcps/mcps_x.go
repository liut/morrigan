package mcps

// IsRemote 判断是否为远程传输类型（SSE 或 Streamable）
func (t TransType) IsRemote() bool {
	return t == TransTypeSSE || t == TransTypeStreamable
}
