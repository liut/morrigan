package tools

import (
	"context"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/liut/morrigan/pkg/services/stores"
	"github.com/liut/morrigan/pkg/settings"
)

type Invoker func(ctx context.Context, params map[string]any) (mcp.Content, error)

type Registry struct {
	sto      stores.Storage
	tools    []mcp.Tool
	invokers map[string]Invoker

	// 受限工具列表（需要 keeper 角色）
	privTools []mcp.Tool
}

func NewRegistry(sto stores.Storage) *Registry {
	r := &Registry{
		sto:      sto,
		tools:    make([]mcp.Tool, 0),
		invokers: make(map[string]Invoker),
	}
	r.initTools()
	return r
}

func (r *Registry) Tools() []mcp.Tool {
	return r.tools
}

func (r *Registry) Invoke(ctx context.Context, name string, params map[string]any) (mcp.Content, error) {
	if name == "" {
		return mcp.NewTextContent("tool name is empty"), nil
	}
	logger().Debugw("invoking", "toolName", name, "params", params)
	for key, invoker := range r.invokers {
		if strings.EqualFold(key, name) {
			return invoker(ctx, params)
		}
	}
	return mcp.NewTextContent("tool not found"), nil
}

func (r *Registry) initTools() {
	// Add KB tools
	if r.sto != nil {
		// 公开工具：KBSearch
		r.tools = append(r.tools,
			mcp.NewTool(ToolNameKBSearch,
				mcp.WithDescription("Search documents in knowledge base with keywords or subject"),
				mcp.WithString("subject", mcp.Required(), mcp.Description("text of keywords or subject")),
			),
		)
		r.invokers[ToolNameKBSearch] = r.callKBSearch

		// 受限工具：KBCreate (需要 keeper 角色)
		r.privTools = append(r.privTools,
			mcp.NewTool(ToolNameKBCreate,
				mcp.WithDescription("Create new document of knowledge base, all parameters are required. Note: Unless the user explicitly requests supplementary content, do not invoke it. Before invoking, always perform a kb_search to confirm there is no corresponding subject or content. If similar content already exists, do not invoke even if requested by the user!"),
				mcp.WithString("title", mcp.Required(), mcp.Description("title of document, like a main name or topic")),
				mcp.WithString("heading", mcp.Required(), mcp.Description("heading of document, like a sub name or property")),
				mcp.WithString("content", mcp.Required(), mcp.Description("long text of content of document")),
			),
		)
		r.invokers[ToolNameKBCreate] = r.callKBCreate
	}

	// 公开工具：Fetch
	r.tools = append(r.tools,
		mcp.NewTool(ToolNameFetch,
			mcp.WithDescription("Fetches a URL from the internet and optionally extracts its contents as markdown"),
			mcp.WithString("url",
				mcp.Required(),
				mcp.Description("URL to fetch"),
			),
			mcp.WithNumber("max_length",
				mcp.DefaultNumber(5000),
				mcp.Description("Maximum number of characters to return, default 5000"),
				mcp.Min(0),
				mcp.Max(1000000),
			),
			mcp.WithNumber("start_index",
				mcp.DefaultNumber(0),
				mcp.Description("On return output starting at this character index, default 0"),
				mcp.Min(0),
			),
			mcp.WithBoolean("raw",
				mcp.DefaultBool(false),
				mcp.Description("Get the actual HTML content without simplification, dfault false"),
			),
		),
	)
	r.invokers[ToolNameFetch] = r.callFetch

	logger().Debugw("init tools", "tools", r.tools, "priv", len(r.privTools))
}

// ToolsFor 返回适合当前上下文的工具列表
// 如果用户有 keeper 角色，返回所有工具；否则只返回公开工具
func (r *Registry) ToolsFor(ctx context.Context) []mcp.Tool {
	if r.IsKeeper(ctx) {
		// 合并公开工具和受限工具
		return append(r.tools, r.privTools...)
	}
	return r.tools
}

// IsKeeper 检查当前上下文中的用户是否具有 keeper 角色
func (r *Registry) IsKeeper(ctx context.Context) bool {
	user, ok := stores.UserFromContext(ctx)
	if !ok {
		return false
	}
	return slices.Contains(user.Roles, settings.Current.KeeperRole)
}
