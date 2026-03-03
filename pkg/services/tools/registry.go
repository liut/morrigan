package tools

import (
	"context"
	"strings"

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
}

func NewRegistry(sto stores.Storage) *Registry {
	r := &Registry{
		sto:      sto,
		tools:    make([]mcps.ToolDescriptor, 0),
		invokers: make(map[string]Invoker),
	}
	r.initTools()
	return r
}

func (r *Registry) Tools() []mcps.ToolDescriptor {
	return r.tools
}

func (r *Registry) Invoke(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	if name == "" {
		return mcps.BuildToolErrorResult("tool name is empty"), nil
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
	if stores.IsKeeper(ctx) {
		// 合并公开工具和受限工具
		return append(r.tools, r.privTools...)
	}
	return r.tools
}
