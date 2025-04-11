package mcputils

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// MCPToolsToOpenAITools 将 mcp.Tool 列表转换为 openai.Tool 列表
func MCPToolsToOpenAITools(tools []mcp.Tool) ([]openai.Tool, error) {
	var openaiTools []openai.Tool
	for _, tool := range tools {
		openaiTool, err := MCPToolToOpenAITool(tool)
		if err != nil {
			return nil, err
		}
		openaiTools = append(openaiTools, openaiTool)
	}
	return openaiTools, nil
}

// MCPToolToOpenAITool 将 mcp.Tool 转换为 openai.Tool
func MCPToolToOpenAITool(tool mcp.Tool) (openai.Tool, error) {
	paramsDef := jsonschema.Definition{
		Type:     jsonschema.Object, // OpenAI Function 的 Parameters 类型必须是 "object"
		Required: tool.InputSchema.Required,
	}
	openaiTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters: jsonschema.Definition{
				Type:     jsonschema.Object, // OpenAI Function 的 Parameters 类型必须是 "object"
				Required: tool.InputSchema.Required,
			},
		},
	}

	if tool.RawInputSchema != nil {
		var parameter jsonschema.Definition
		if err := json.Unmarshal(tool.RawInputSchema, &parameter); err != nil {
			logger().Infow("unmarshal input schema fail", "raw", tool.RawInputSchema, "err", err)
			return openaiTool, err
		}
		openaiTool.Function.Parameters = parameter
	} else {
		parameters := make(map[string]jsonschema.Definition)
		for paramName, paramDef := range tool.InputSchema.Properties {
			if properties, ok := paramDef.(map[string]any); ok {
				parameters[paramName] = propertiesToDefinition(properties)
			}
		}
		paramsDef.Properties = parameters
		openaiTool.Function.Parameters = paramsDef
	}

	return openaiTool, nil
}

func propertiesToDefinition(properties map[string]any) jsonschema.Definition {
	// 使用结构化的 InputSchema
	output := jsonschema.Definition{
		Type: jsonschema.Object, // 默认为对象类型
	}

	for propName, propDef := range properties {
		logger().Debugw("propertiesToDefinition", "propName", propName,
			"propDef", propDef)
		switch propName {
		case "type":
			if typeStr, ok := propDef.(string); ok {
				output.Type = jsonschema.DataType(typeStr)
			}
		case "description":
			if descStr, ok := propDef.(string); ok {
				output.Description = descStr
			}
		case "enum":
			// 处理枚举值
			output.Enum = convertToStringSlice(propDef)
		case "properties":
			// 处理属性对象
			logger().Debugw("propertiesToDefinition", "propName", propName, "propDef", propDef)
			output.Properties = make(map[string]jsonschema.Definition)
			if propsMap, ok := propDef.(map[string]any); ok {
				for key, subprop := range propsMap {
					if subPropMap, ok := subprop.(map[string]any); ok {
						output.Properties[key] = propertiesToDefinition(subPropMap)
					}
				}
			}
		case "required":
			// 处理必需字段
			output.Required = convertToStringSlice(propDef)
		case "items":
			// 处理数组项
			if itemsMap, ok := propDef.(map[string]any); ok {
				itemsDef := propertiesToDefinition(itemsMap)
				output.Items = &itemsDef
			}
		case "additionalProperties":
			// 处理额外属性
			output.AdditionalProperties = propDef
		}
	}

	return output
}

// convertToStringSlice 将不同类型的列表转换为字符串切片
func convertToStringSlice(value any) (result []string) {
	if strSlice, ok := value.([]string); ok {
		return strSlice
	} else if anySlice, ok := value.([]any); ok {
		for _, val := range anySlice {
			if strVal, ok := val.(string); ok {
				result = append(result, strVal)
			}
		}
	}

	return
}
