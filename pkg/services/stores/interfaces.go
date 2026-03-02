package stores

type Storage interface {
	Cob() CobStore     // gened
	MCP() MCPStore     // gened
	Convo() ConvoStore // gened
}
