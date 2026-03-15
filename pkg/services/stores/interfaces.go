package stores

// Storage is the storage interface defining access methods for various data stores
type Storage interface {
	Corpus() CorpuStore // gened
	KB() CorpuStore     // alias
	MCP() MCPStore      // gened
	Convo() ConvoStore  // gened
}
