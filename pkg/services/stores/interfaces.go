package stores

type Storage interface {
	Qa() QaStore   // gened
	MCP() MCPStore // gened
}
