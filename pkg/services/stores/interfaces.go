package stores

// Storage is the storage interface defining access methods for various data stores
type Storage interface {
	Cob() CobStore     // gened
	MCP() MCPStore     // gened
	Convo() ConvoStore // gened
}
