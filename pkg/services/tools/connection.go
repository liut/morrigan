package tools

import (
	"fmt"

	"github.com/mark3labs/mcp-go/client"

	"github.com/liut/morign/pkg/models/mcps"
)

// MCPConnection 表示一个到 MCP 服务器的连接
type MCPConnection struct {
	Name      string
	URL       string
	TransType mcps.TransType
	client    *client.Client
	toolNames []string // 注册的工具名列表
}

func (mcpc *MCPConnection) getToolKey(name string) string {
	return fmt.Sprintf("%s:%s", mcpc.Name, name)
}
