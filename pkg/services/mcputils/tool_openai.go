package mcputils

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/liut/morrigan/pkg/models/mcps"
)

// MCPToolsToOpenAITools 将 mcps.ToolDescriptor 列表转换为 openai.Tool 列表
func MCPToolsToOpenAITools(tools []mcps.ToolDescriptor) ([]openai.Tool, error) {
	openaiTools := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		openaiTool, err := MCPToolToOpenAITool(tool)
		if err != nil {
			return nil, err
		}
		openaiTools = append(openaiTools, openaiTool)
	}
	return openaiTools, nil
}

// MCPToolToOpenAITool 将 mcps.ToolDescriptor 转换为 openai.Tool
func MCPToolToOpenAITool(tool mcps.ToolDescriptor) (openai.Tool, error) {
	openaiTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        tool.Name,
			Description: tool.Description,
		},
	}

	// 将 InputSchema 转换为 openai 的格式
	if tool.InputSchema != nil {
		paramsDef := convertInputSchemaToDefinition(tool.InputSchema)
		openaiTool.Function.Parameters = paramsDef
	}

	return openaiTool, nil
}

// convertInputSchemaToDefinition 将 map[string]any 格式的 InputSchema 转换为 jsonschema.Definition
func convertInputSchemaToDefinition(schema map[string]any) jsonschema.Definition {
	def := jsonschema.Definition{
		Type: jsonschema.Object,
	}

	// 处理 type
	if t, ok := schema["type"].(string); ok {
		def.Type = jsonschema.DataType(t)
	}

	// 处理 required - 可能是 []any 或 []string
	if required, ok := schema["required"]; ok {
		switch r := required.(type) {
		case []any:
			for _, item := range r {
				if rs, ok := item.(string); ok {
					def.Required = append(def.Required, rs)
				}
			}
		case []string:
			def.Required = r
		}
	}

	// 处理 properties
	if properties, ok := schema["properties"].(map[string]any); ok {
		def.Properties = make(map[string]jsonschema.Definition)
		for propName, propDef := range properties {
			if propMap, ok := propDef.(map[string]any); ok {
				def.Properties[propName] = convertPropertyToDefinition(propMap)
			}
		}
	}

	return def
}

// convertPropertyToDefinition 将属性定义转换为 jsonschema.Definition
func convertPropertyToDefinition(prop map[string]any) jsonschema.Definition {
	def := jsonschema.Definition{}

	// 处理 type
	if t, ok := prop["type"].(string); ok {
		def.Type = jsonschema.DataType(t)
	}

	// 处理 description
	if desc, ok := prop["description"].(string); ok {
		def.Description = desc
	}

	// 处理 enum - 可能是 []any 或 []string
	if enum, ok := prop["enum"]; ok {
		switch e := enum.(type) {
		case []any:
			def.Enum = toStringSlice(e)
		case []string:
			def.Enum = e
		}
	}

	// 处理 items (数组类型)
	if items, ok := prop["items"].(map[string]any); ok {
		itemDef := convertPropertyToDefinition(items)
		def.Items = &itemDef
	}

	// 处理 properties (嵌套对象)
	if properties, ok := prop["properties"].(map[string]any); ok {
		def.Properties = make(map[string]jsonschema.Definition)
		for propName, propDef := range properties {
			if propMap, ok := propDef.(map[string]any); ok {
				def.Properties[propName] = convertPropertyToDefinition(propMap)
			}
		}
	}

	return def
}

// toStringSlice 将 []any 转换为 []string
func toStringSlice(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
