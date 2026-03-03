package mcputils

import (
	"encoding/json"
	"testing"

	"github.com/liut/morrigan/pkg/models/mcps"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPToolToOpenAITool(t *testing.T) {
	// 构造一个复杂的 MCP Tool 进行测试
	mcpTool := mcps.ToolDescriptor{
		Name:        "calculate",
		Description: "Perform basic arithmetic calculations",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{
					"type":        "string",
					"description": "The arithmetic operation to perform (add, subtract, multiply, divide)",
					"enum":        []string{"add", "subtract", "multiply", "divide"},
				},
				"x": map[string]any{
					"type":        "number",
					"description": "First number",
				},
				"y": map[string]any{
					"type":        "number",
					"description": "Second number",
				},
			},
			"required": []string{"operation", "x", "y"},
		},
	}

	// 将 MCP Tool 转换为 OpenAI Tool
	openaiTool, err := MCPToolToOpenAITool(mcpTool)
	require.NoError(t, err)

	// 验证基本属性
	assert.Equal(t, openai.ToolTypeFunction, openaiTool.Type)
	assert.Equal(t, "calculate", openaiTool.Function.Name)
	assert.Equal(t, "Perform basic arithmetic calculations", openaiTool.Function.Description)

	// 验证 parameters 属性
	params := openaiTool.Function.Parameters
	assert.NotNil(t, params)

	// 将 parameters 转换为 JSON 字符串，便于验证内容
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)
	// t.Logf("paramsJSON: %s", paramsJSON)

	// 解析回结构体以便验证具体字段
	var paramsDef struct {
		Type       string   `json:"type"`
		Required   []string `json:"required"`
		Properties struct {
			Operation struct {
				Type        string   `json:"type"`
				Description string   `json:"description"`
				Enum        []string `json:"enum"`
			} `json:"operation"`
			X struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"x"`
			Y struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"y"`
		} `json:"properties"`
	}
	err = json.Unmarshal(paramsJSON, &paramsDef)
	require.NoError(t, err)

	// 验证详细属性
	assert.Equal(t, "object", paramsDef.Type)
	assert.Contains(t, paramsDef.Required, "operation")
	assert.Contains(t, paramsDef.Required, "x")
	assert.Contains(t, paramsDef.Required, "y")

	// 验证 operation 属性
	assert.Equal(t, "string", paramsDef.Properties.Operation.Type)
	assert.Equal(t, "The arithmetic operation to perform (add, subtract, multiply, divide)", paramsDef.Properties.Operation.Description)
	assert.ElementsMatch(t, []string{"add", "subtract", "multiply", "divide"}, paramsDef.Properties.Operation.Enum)

	// 验证 x 和 y 属性
	assert.Equal(t, "number", paramsDef.Properties.X.Type)
	assert.Equal(t, "First number", paramsDef.Properties.X.Description)
	assert.Equal(t, "number", paramsDef.Properties.Y.Type)
	assert.Equal(t, "Second number", paramsDef.Properties.Y.Description)
}

// 测试使用 RawInputSchema 的转换
func TestMCPToolToOpenAIToolWithRawSchema(t *testing.T) {
	// 准备一个 JSON Schema
	rawSchema := map[string]any{
		"type": "object",
		"required": []string{"name", "age"},
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "User name",
			},
			"age": map[string]any{
				"type":        "number",
				"description": "User age",
			},
		},
	}

	// 创建带有 RawInputSchema 的 MCP Tool
	tool := mcps.ToolDescriptor{
		Name:        "createUser",
		Description: "Create a new user",
		InputSchema: rawSchema,
	}

	// 转换为 OpenAI Tool
	openaiTool, err := MCPToolToOpenAITool(tool)
	require.NoError(t, err)

	// 验证基本属性
	assert.Equal(t, "createUser", openaiTool.Function.Name)
	assert.Equal(t, "Create a new user", openaiTool.Function.Description)

	// 验证 parameters
	params := openaiTool.Function.Parameters
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	var paramsDef map[string]any
	err = json.Unmarshal(paramsJSON, &paramsDef)
	require.NoError(t, err)

	// 验证从 RawInputSchema 正确解析
	assert.Equal(t, "object", paramsDef["type"])
	requiredProps, ok := paramsDef["required"].([]any)
	assert.True(t, ok)
	assert.Contains(t, requiredProps, "name")
	assert.Contains(t, requiredProps, "age")
}

// 测试嵌套对象
func TestMCPToolToOpenAIToolWithNestedObject(t *testing.T) {
	// 使用 map 创建嵌套对象
	addressProps := map[string]any{
		"type":        "object",
		"description": "Person's address",
		"properties": map[string]any{
			"street": map[string]any{
				"type":        "string",
				"description": "Street name",
			},
			"city": map[string]any{
				"type":        "string",
				"description": "City name",
			},
			"zipcode": map[string]any{
				"type":        "string",
				"description": "ZIP code",
			},
		},
	}

	personProps := map[string]any{
		"name": map[string]any{
			"type":        "string",
			"description": "Person's name",
		},
		"address": addressProps,
	}

	mcpTool := mcps.ToolDescriptor{
		Name:        "createPerson",
		Description: "Create a new person record",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"person": map[string]any{
					"type":        "object",
					"description": "Person information",
					"properties":  personProps,
				},
			},
			"required": []string{"person"},
		},
	}

	openaiTool, err := MCPToolToOpenAITool(mcpTool)
	require.NoError(t, err)

	// 验证嵌套结构
	params := openaiTool.Function.Parameters
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)
	// t.Logf("paramsJSON: %s", paramsJSON)

	var paramsDef map[string]any
	err = json.Unmarshal(paramsJSON, &paramsDef)
	require.NoError(t, err)

	properties, ok := paramsDef["properties"].(map[string]any)
	assert.True(t, ok)

	personProp, ok := properties["person"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "object", personProp["type"])

	personProperties, ok := personProp["properties"].(map[string]any)
	assert.True(t, ok)

	addressProp, ok := personProperties["address"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "object", addressProp["type"])

	addressProperties, ok := addressProp["properties"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, addressProperties, "street")
	assert.Contains(t, addressProperties, "city")
	assert.Contains(t, addressProperties, "zipcode")
}

// 测试数组类型
func TestMCPToolToOpenAIToolWithArray(t *testing.T) {
	mcpTool := mcps.ToolDescriptor{
		Name:        "addTags",
		Description: "Add tags to an item",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"itemId": map[string]any{
					"type":        "string",
					"description": "ID of the item",
				},
				"tags": map[string]any{
					"type":        "array",
					"description": "List of tags",
					"items": map[string]any{
						"type": "string",
						"enum": []string{"red", "green", "blue"},
					},
				},
			},
			"required": []string{"itemId", "tags"},
		},
	}

	openaiTool, err := MCPToolToOpenAITool(mcpTool)
	require.NoError(t, err)

	// 验证数组类型
	params := openaiTool.Function.Parameters
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	var paramsDef map[string]any
	err = json.Unmarshal(paramsJSON, &paramsDef)
	require.NoError(t, err)

	properties, ok := paramsDef["properties"].(map[string]any)
	assert.True(t, ok)

	tagsProp, ok := properties["tags"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "array", tagsProp["type"])

	// 验证数组项类型
	// t.Logf("tagsProp: %+v", tagsProp)
	items, ok := tagsProp["items"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "string", items["type"])
}

// 测试布尔类型
func TestMCPToolToOpenAIToolWithBoolean(t *testing.T) {
	mcpTool := mcps.ToolDescriptor{
		Name:        "setUserStatus",
		Description: "Set user active status",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"userId": map[string]any{
					"type":        "string",
					"description": "ID of the user",
				},
				"isActive": map[string]any{
					"type":        "boolean",
					"description": "Whether the user is active",
				},
			},
			"required": []string{"userId", "isActive"},
		},
	}

	openaiTool, err := MCPToolToOpenAITool(mcpTool)
	require.NoError(t, err)

	// 验证布尔类型
	params := openaiTool.Function.Parameters
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	var paramsDef map[string]any
	err = json.Unmarshal(paramsJSON, &paramsDef)
	require.NoError(t, err)

	properties, ok := paramsDef["properties"].(map[string]any)
	assert.True(t, ok)

	isActiveProp, ok := properties["isActive"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "boolean", isActiveProp["type"])
}

// 测试无参数工具
func TestMCPToolToOpenAIToolWithNoParams(t *testing.T) {
	mcpTool := mcps.ToolDescriptor{
		Name:        "ping",
		Description: "Simple ping command with no parameters",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}

	openaiTool, err := MCPToolToOpenAITool(mcpTool)
	require.NoError(t, err)

	// 验证无参数处理
	params := openaiTool.Function.Parameters
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	var paramsDef map[string]any
	err = json.Unmarshal(paramsJSON, &paramsDef)
	require.NoError(t, err)

	// 即使没有参数，应该有 type: object
	assert.Equal(t, "object", paramsDef["type"])
	properties, ok := paramsDef["properties"].(map[string]any)
	if ok {
		assert.Empty(t, properties)
	}
}

// 测试工具列表转换函数
func TestMCPToolsToOpenAITools(t *testing.T) {
	// 创建多个工具
	tool1 := mcps.ToolDescriptor{
		Name:        "tool1",
		Description: "Tool 1",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"param1": map[string]any{
					"type":        "string",
					"description": "Parameter 1",
				},
			},
		},
	}

	tool2 := mcps.ToolDescriptor{
		Name:        "tool2",
		Description: "Tool 2",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"param2": map[string]any{
					"type":        "number",
					"description": "Parameter 2",
				},
			},
		},
	}

	tools := []mcps.ToolDescriptor{tool1, tool2}

	// 转换工具列表
	openaiTools, err := MCPToolsToOpenAITools(tools)
	require.NoError(t, err)

	// 验证转换结果
	assert.Len(t, openaiTools, 2)
	assert.Equal(t, "tool1", openaiTools[0].Function.Name)
	assert.Equal(t, "tool2", openaiTools[1].Function.Name)
}
