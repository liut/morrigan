package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcp "github.com/mark3labs/mcp-go/mcp"

	"github.com/liut/morrigan/pkg/models/mcps"
	"github.com/liut/morrigan/pkg/services/stores"
)

type Invoker func(ctx context.Context, params map[string]any) (map[string]any, error)

type Registry struct {
	sto      stores.Storage
	tools    []mcps.ToolDescriptor
	invokers map[string]Invoker

	// 受限工具列表（需要 keeper 角色）
	privTools []mcps.ToolDescriptor

	// OAuth MCP 相关
	oauthMCPEnabled bool
	oauthMCPInited  bool // 是否已初始化工具列表
	oauthGetToken   func(ctx context.Context) string
	oauthEndpoint   string
	oauthClients    map[string]*client.Client // token -> client 缓存
	oauthClientsMu  sync.Mutex
	clientInfo      mcp.Implementation // MCP 客户端信息
}

// RegistryOption 用于配置 Registry 的可选参数
type RegistryOption func(*Registry)

// WithClientInfo 设置 MCP 客户端信息
func WithClientInfo(name, version string) RegistryOption {
	return func(r *Registry) {
		r.clientInfo = mcp.Implementation{Name: name, Version: version}
	}
}

// WithOAuthMCP 配置 OAuth MCP
func WithOAuthMCP(endpoint string, getToken func(ctx context.Context) string) RegistryOption {
	return func(r *Registry) {
		if endpoint != "" && getToken != nil {
			r.oauthEndpoint = endpoint
			r.oauthGetToken = getToken
			r.oauthMCPEnabled = true
			logger().Infow("OAuth MCP configured", "endpoint", endpoint)
		}
	}
}

// NewRegistry 创建工具注册表
func NewRegistry(sto stores.Storage, opts ...RegistryOption) *Registry {
	r := &Registry{
		sto:          sto,
		tools:        make([]mcps.ToolDescriptor, 0),
		invokers:     make(map[string]Invoker),
		oauthClients: make(map[string]*client.Client),
	}
	r.initTools()

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Registry) Tools() []mcps.ToolDescriptor {
	// 注意：OAuth MCP 工具列表延迟初始化，工具列表会在 Invoke/ToolsFor 时获取
	return r.tools
}

func (r *Registry) Invoke(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	if name == "" {
		return mcps.BuildToolErrorResult("tool name is empty"), nil
	}

	// 延迟初始化 OAuth MCP 工具列表
	if r.oauthMCPEnabled && !r.oauthMCPInited {
		if err := r.initOAuthMCPTools(ctx); err != nil {
			logger().Warnw("failed to init OAuth MCP tools", "err", err)
		}
	}

	logger().Debugw("invoking", "toolName", name, "params", params)
	for key, invoker := range r.invokers {
		if strings.EqualFold(key, name) {
			return invoker(ctx, params)
		}
	}
	return mcps.BuildToolErrorResult("tool not found"), nil
}

func (r *Registry) initTools() {
	// Add KB tools
	if r.sto != nil {
		// 公开工具：KBSearch
		r.tools = append(r.tools, mcps.ToolDescriptor{
			Name:        ToolNameKBSearch,
			Description: "Search documents in knowledge base with keywords or subject",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"subject": map[string]any{
						"type":        "string",
						"description": "text of keywords or subject",
					},
				},
				"required": []string{"subject"},
			},
		})
		r.invokers[ToolNameKBSearch] = r.callKBSearch

		// 受限工具：KBCreate (需要 keeper 角色)
		r.privTools = append(r.privTools, mcps.ToolDescriptor{
			Name:        ToolNameKBCreate,
			Description: "Create new document of knowledge base, all parameters are required. Note: Unless the user explicitly requests supplementary content, do not invoke it. Before invoking, always perform a kb_search to confirm there is no corresponding subject or content. If similar content already exists, do not invoke even if requested by the user!",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "title of document, like a main name or topic",
					},
					"heading": map[string]any{
						"type":        "string",
						"description": "heading of document, like a sub name or property",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "long text of content of document",
					},
				},
				"required": []string{"title", "heading", "content"},
			},
		})
		r.invokers[ToolNameKBCreate] = r.callKBCreate
	}

	// 公开工具：Fetch
	r.tools = append(r.tools, mcps.ToolDescriptor{
		Name:        ToolNameFetch,
		Description: "Fetches a URL from the internet and optionally extracts its contents as markdown",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "URL to fetch",
				},
				"max_length": map[string]any{
					"type":        "number",
					"description": "Maximum number of characters to return, default 5000",
					"default":     5000,
					"minimum":     0,
					"maximum":     1000000,
				},
				"start_index": map[string]any{
					"type":        "number",
					"description": "On return output starting at this character index, default 0",
					"default":     0,
					"minimum":     0,
				},
				"raw": map[string]any{
					"type":        "boolean",
					"description": "Get the actual HTML content without simplification, default false",
					"default":     false,
				},
			},
			"required": []string{"url"},
		},
	})
	r.invokers[ToolNameFetch] = r.callFetch

	logger().Debugw("init tools", "tools", r.tools, "priv", len(r.privTools))
}

// ToolsFor 返回适合当前上下文的工具列表
// 如果用户有 keeper 角色，返回所有工具；否则只返回公开工具
func (r *Registry) ToolsFor(ctx context.Context) []mcps.ToolDescriptor {
	// 延迟初始化 OAuth MCP 工具列表（当有 context 时尝试获取）
	if r.oauthMCPEnabled && !r.oauthMCPInited {
		if err := r.initOAuthMCPTools(ctx); err != nil {
			logger().Warnw("failed to init OAuth MCP tools", "err", err)
		}
	}

	if stores.IsKeeper(ctx) {
		// 合并公开工具和受限工具
		return append(r.tools, r.privTools...)
	}
	return r.tools
}

// InitOAuthMCP 初始化 OAuth MCP 连接（延迟获取工具列表）
// initOAuthMCPTools 延迟初始化 OAuth MCP 工具列表
// 只有在有有效 token 时才能获取工具列表
func (r *Registry) initOAuthMCPTools(ctx context.Context) error {
	if r.oauthMCPInited {
		return nil
	}

	// 检查是否有 token
	if r.oauthGetToken == nil {
		return nil
	}
	tok := r.oauthGetToken(ctx)
	if tok == "" {
		logger().Infow("no token available for OAuth MCP, skipping tool list fetch")
		return nil
	}

	logger().Infow("fetching OAuth MCP tools", "endpoint", r.oauthEndpoint)

	// 使用 getOAuthClient 获取已初始化好的 client
	c, err := r.getOAuthClient(ctx)
	if err != nil {
		logger().Warnw("failed to get OAuth MCP client", "err", err)
		return err
	}
	if c == nil {
		logger().Infow("no OAuth MCP client available")
		return nil
	}

	// 获取工具列表
	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		logger().Warnw("failed to list OAuth MCP tools", "err", err)
		return err
	}

	logger().Infow("OAuth MCP tools fetched", "count", len(result.Tools), "tools", func() []string {
		names := make([]string, len(result.Tools))
		for i, t := range result.Tools {
			names[i] = t.Name
		}
		return names
	}())

	// 转换为本地 ToolDescriptor 并注册 invoker
	for _, tool := range result.Tools {
		// InputSchema 是 ToolInputSchema 类型，需要转换
		inputSchema := convertInputSchema(tool.InputSchema)
		r.tools = append(r.tools, mcps.ToolDescriptor{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: inputSchema,
		})
		r.invokers[tool.Name] = func(ctx context.Context, params map[string]any) (map[string]any, error) {
			return r.callOAuthTool(ctx, tool.Name, params)
		}
	}

	r.oauthMCPInited = true

	logger().Info("OAuth MCP tools initialized")
	return nil
}

// getOAuthClient 获取或创建对应 token 的 MCP client
func (r *Registry) getOAuthClient(ctx context.Context) (*client.Client, error) {
	if r.oauthGetToken == nil {
		return nil, nil
	}
	tok := r.oauthGetToken(ctx)
	if tok == "" {
		return nil, nil
	}

	// 加锁检查是否有缓存的 client
	r.oauthClientsMu.Lock()
	defer r.oauthClientsMu.Unlock()

	if c, ok := r.oauthClients[tok]; ok {
		return c, nil
	}

	// 创建新的 client
	tp, err := transport.NewStreamableHTTP(r.oauthEndpoint,
		transport.WithHTTPHeaderFunc(func(ctx context.Context) map[string]string {
			if t := r.oauthGetToken(ctx); t != "" {
				return map[string]string{"Authorization": "Bearer " + t}
			}
			return nil
		}))
	if err != nil {
		return nil, err
	}

	c := client.NewClient(tp)
	if err := c.Start(ctx); err != nil {
		return nil, err
	}

	// Initialize MCP 协议
	if _, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      r.clientInfo,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize OAuth MCP: %w", err)
	}

	r.oauthClients[tok] = c
	logger().Debugw("created new OAuth MCP client", "token", tok[:min(8, len(tok))])
	return c, nil
}

// callOAuthTool 调用 OAuth MCP 工具
func (r *Registry) callOAuthTool(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	c, err := r.getOAuthClient(ctx)
	if err != nil {
		logger().Errorw("failed to get OAuth MCP client", "err", err)
		return mcps.BuildToolErrorResult(err.Error()), nil
	}
	if c == nil {
		return mcps.BuildToolErrorResult("no OAuth token available"), nil
	}

	// 确保 params 不为空，避免 "empty arguments" 错误
	if params == nil {
		params = make(map[string]any)
	}

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: params,
		},
	})
	if err != nil {
		logger().Errorw("OAuth MCP tool call failed", "tool", name, "err", err)
		return mcps.BuildToolErrorResult(err.Error()), nil
	}

	// 转换 MCP 结果为本地格式
	return convertMCPToolResult(result), nil
}

// convertInputSchema 将 ToolInputSchema 转换为 map[string]any
func convertInputSchema(schema mcp.ToolInputSchema) map[string]any {
	return map[string]any{
		"type":       schema.Type,
		"properties": schema.Properties,
		"required":   schema.Required,
	}
}

// convertMCPToolResult 将 MCP 工具结果转换为本地格式
func convertMCPToolResult(result *mcp.CallToolResult) map[string]any {
	if len(result.Content) == 0 {
		return mcps.BuildToolSuccessResult(nil)
	}

	// 取第一个 content
	content := result.Content[0]
	if textContent, ok := content.(mcp.TextContent); ok {
		if result.IsError {
			return mcps.BuildToolErrorResult(textContent.Text)
		}
		return mcps.BuildToolSuccessResult(map[string]any{
			"text": textContent.Text,
		})
	}

	// 其他类型直接返回
	return mcps.BuildToolSuccessResult(map[string]any{
		"content": content,
	})
}
