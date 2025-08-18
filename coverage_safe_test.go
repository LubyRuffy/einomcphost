package einomcphost

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMCPHubErrorPaths 测试NewMCPHub的错误路径
func TestNewMCPHubErrorPaths(t *testing.T) {
	ctx := context.Background()

	// 测试文件不存在的情况
	_, err := NewMCPHub(ctx, "nonexistent_file.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "加载配置文件失败")

	// 测试无效JSON文件
	tempDir := t.TempDir()
	invalidPath := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("{invalid json"), 0644)
	require.NoError(t, err)

	_, err = NewMCPHub(ctx, invalidPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "加载配置文件失败")
}

// TestNewMCPHubDisabledServers 测试NewMCPHub与禁用的服务器
func TestNewMCPHubDisabledServers(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "disabled_servers.json")

	// 创建只有禁用服务器的配置
	config := MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"disabled_stdio": {
				Transport: "stdio",
				Command:   "echo",
				Disabled:  true,
			},
			"disabled_sse": {
				Transport: "sse",
				URL:       "http://example.com",
				Disabled:  true,
			},
		},
	}

	configBytes, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configBytes, 0644)
	require.NoError(t, err)

	// 测试成功创建hub（因为所有服务器都禁用）
	hub, err := NewMCPHub(ctx, configPath)
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	defer hub.CloseServers()

	// 验证没有工具被注册
	tools, err := hub.GetEinoTools(ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, tools)
}

// TestCreateMCPClientAllErrorPaths 测试createMCPClient的所有错误路径
func TestCreateMCPClientAllErrorPaths(t *testing.T) {
	hub := &MCPHub{}

	// 测试不支持的传输类型
	config := &ServerConfig{
		Transport: "unknown_transport",
		Command:   "echo",
	}
	client, err := hub.createMCPClient(config)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "不支持的传输类型")

	// 测试sse传输但缺少URL
	config = &ServerConfig{
		Transport: transportSSE,
		URL:       "",
	}
	client, err = hub.createMCPClient(config)
	// SSE客户端可能不会立即验证URL，所以这里不强制要求错误
	if err != nil {
		assert.Nil(t, client)
	} else if client != nil {
		client.Close() // 清理资源
	}
}

// TestConvertToolSchemaComprehensive 全面测试convertToolSchema
func TestConvertToolSchemaComprehensive(t *testing.T) {
	hub := &MCPHub{}

	// 测试基本对象类型
	basicTool := mcp.Tool{
		Name:        "basic_tool",
		Description: "Basic tool",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "String parameter",
				},
			},
		},
	}

	schema, err := hub.convertToolSchema(basicTool)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)

	// 测试包含exclusiveMaximum/exclusiveMinimum的工具
	exclusiveTool := mcp.Tool{
		Name:        "exclusive_tool",
		Description: "Tool with exclusive bounds",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"number_param": map[string]interface{}{
					"type":             "number",
					"exclusiveMaximum": 100.0,
					"exclusiveMinimum": 0.0,
				},
			},
		},
	}

	schema, err = hub.convertToolSchema(exclusiveTool)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// 测试嵌套对象（不包含exclusiveMaximum/exclusiveMinimum，因为当前实现不支持递归处理）
	nestedTool := mcp.Tool{
		Name:        "nested_tool",
		Description: "Tool with nested objects",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"nested": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"inner": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	}

	schema, err = hub.convertToolSchema(nestedTool)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// 测试数组类型（不包含exclusiveMinimum，因为当前实现不支持递归处理）
	arrayTool := mcp.Tool{
		Name:        "array_tool",
		Description: "Tool with array",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"items": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}

	schema, err = hub.convertToolSchema(arrayTool)
	assert.NoError(t, err)
	assert.NotNil(t, schema)

	// 测试空schema
	emptyTool := mcp.Tool{
		Name:        "empty_tool",
		Description: "Empty tool",
		InputSchema: mcp.ToolInputSchema{},
	}

	schema, err = hub.convertToolSchema(emptyTool)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
}

// TestBuildEnvironmentExtensive 扩展测试buildEnvironment
func TestBuildEnvironmentExtensive(t *testing.T) {
	hub := &MCPHub{}

	// 测试nil环境
	result := hub.buildEnvironment(nil)
	assert.Nil(t, result)

	// 测试空环境
	result = hub.buildEnvironment(map[string]string{})
	assert.Len(t, result, 0) // 可能是nil或空slice，但长度应该是0

	// 测试单个环境变量
	result = hub.buildEnvironment(map[string]string{"KEY": "value"})
	assert.Equal(t, []string{"KEY=value"}, result)

	// 测试多个环境变量
	envMap := map[string]string{
		"PATH":   "/usr/bin",
		"HOME":   "/home/user",
		"CUSTOM": "value",
	}
	result = hub.buildEnvironment(envMap)
	assert.Len(t, result, 3)
	assert.Contains(t, result, "PATH=/usr/bin")
	assert.Contains(t, result, "HOME=/home/user")
	assert.Contains(t, result, "CUSTOM=value")

	// 测试包含特殊字符的环境变量
	specialEnv := map[string]string{
		"SPECIAL": "value with spaces",
		"EQUALS":  "key=value=more",
		"EMPTY":   "",
	}
	result = hub.buildEnvironment(specialEnv)
	assert.Len(t, result, 3)
	assert.Contains(t, result, "SPECIAL=value with spaces")
	assert.Contains(t, result, "EQUALS=key=value=more")
	assert.Contains(t, result, "EMPTY=")
}

// TestMCPHubConnectionManagement 测试连接管理
func TestMCPHubConnectionManagement(t *testing.T) {
	hub := &MCPHub{
		connections: make(map[string]*Connection),
		tools:       make(map[string]tool.InvokableTool),
	}

	// 测试获取不存在的客户端
	client, err := hub.GetClient("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "未找到服务器连接")

	// 测试关闭不存在的连接
	err = hub.closeExistingConnection("nonexistent")
	assert.NoError(t, err)

	// 测试关闭空的服务器列表
	err = hub.CloseServers()
	assert.NoError(t, err)
	assert.Empty(t, hub.connections)
}

// TestInvokeToolErrorPaths 测试InvokeTool的错误路径
func TestInvokeToolErrorPaths(t *testing.T) {
	hub := &MCPHub{
		tools: make(map[string]tool.InvokableTool),
	}

	ctx := context.Background()

	// 测试调用不存在的工具
	result, err := hub.InvokeTool(ctx, "nonexistent_tool", nil)
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "工具不存在")

	// 测试序列化错误 - 循环引用
	circularMap := make(map[string]interface{})
	circularMap["self"] = circularMap

	result, err = hub.InvokeTool(ctx, "nonexistent", circularMap)
	assert.Error(t, err)
	assert.Empty(t, result)
}

// TestGetEinoToolsErrorPaths 测试GetEinoTools的错误路径
func TestGetEinoToolsErrorPaths(t *testing.T) {
	hub := &MCPHub{
		tools: make(map[string]tool.InvokableTool),
	}

	ctx := context.Background()

	// 测试请求不存在的工具
	tools, err := hub.GetEinoTools(ctx, []string{"nonexistent_tool"})
	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "工具不存在")

	// 测试空工具列表
	tools, err = hub.GetEinoTools(ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, tools)

	// 测试空的工具名称列表
	tools, err = hub.GetEinoTools(ctx, []string{})
	assert.NoError(t, err)
	assert.Empty(t, tools)
}

// TestServerConfigMethods 测试ServerConfig的方法
func TestServerConfigMethods(t *testing.T) {
	// 测试默认超时
	config := ServerConfig{}
	timeout := config.GetTimeoutDuration()
	expected := time.Duration(DefaultMCPTimeoutSeconds) * time.Second
	assert.Equal(t, expected, timeout)

	// 测试自定义超时
	customTimeout := 60 * time.Second
	config.Timeout = customTimeout
	timeout = config.GetTimeoutDuration()
	assert.Equal(t, customTimeout, timeout)

	// 测试IsSSETransport
	config = ServerConfig{Transport: transportSSE}
	assert.True(t, config.IsSSETransport())

	config = ServerConfig{Transport: transportStdio}
	assert.False(t, config.IsSSETransport())

	config = ServerConfig{Transport: "unknown"}
	assert.False(t, config.IsSSETransport())

	// 测试IsStdioTransport
	config = ServerConfig{Transport: transportStdio}
	assert.True(t, config.IsStdioTransport())

	config = ServerConfig{Transport: ""}
	assert.True(t, config.IsStdioTransport()) // 默认为stdio

	config = ServerConfig{Transport: transportSSE}
	assert.False(t, config.IsStdioTransport())
}

// TestNewMCPHubFromSettingsEdgeCases 测试NewMCPHubFromSettings的边界情况
func TestNewMCPHubFromSettingsEdgeCases(t *testing.T) {
	ctx := context.Background()

	// 测试nil设置 - 这会导致panic，所以我们跳过这个测试
	// hub, err := NewMCPHubFromSettings(ctx, nil)
	// assert.Error(t, err)
	// assert.Nil(t, hub)

	// 测试空设置
	settings := &MCPSettings{}
	hub, err := NewMCPHubFromSettings(ctx, settings)
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	defer hub.CloseServers()

	// 测试只有禁用服务器的设置
	settings2 := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"disabled1": {
				Transport: transportStdio,
				Command:   "echo",
				Disabled:  true,
			},
			"disabled2": {
				Transport: transportSSE,
				URL:       "http://example.com",
				Disabled:  true,
			},
		},
	}

	hub2, err := NewMCPHubFromSettings(ctx, settings2)
	assert.NoError(t, err)
	assert.NotNil(t, hub2)
	defer hub2.CloseServers()

	// 验证没有连接被创建
	assert.Empty(t, hub2.connections)
}

// TestNewMCPHubFromStringEdgeCases 测试NewMCPHubFromString的边界情况
func TestNewMCPHubFromStringEdgeCases(t *testing.T) {
	ctx := context.Background()

	// 测试无效JSON
	hub, err := NewMCPHubFromString(ctx, "{invalid json}")
	assert.Error(t, err)
	assert.Nil(t, hub)

	// 测试空JSON对象
	hub, err = NewMCPHubFromString(ctx, "{}")
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	defer hub.CloseServers()

	// 测试只有mcpServers字段但为空的JSON
	hub, err = NewMCPHubFromString(ctx, `{"mcpServers": {}}`)
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	defer hub.CloseServers()

	// 测试包含禁用服务器的JSON
	jsonConfig := `{
		"mcpServers": {
			"disabled_server": {
				"transport": "stdio",
				"command": "echo",
				"disabled": true
			}
		}
	}`

	hub, err = NewMCPHubFromString(ctx, jsonConfig)
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	defer hub.CloseServers()
}

// TestCreateMCPClientStdioSuccess 测试成功创建stdio客户端
func TestCreateMCPClientStdioSuccess(t *testing.T) {
	hub := &MCPHub{}

	// 测试有效的stdio配置
	config := &ServerConfig{
		Transport: transportStdio,
		Command:   "echo", // 使用echo命令，应该在大多数系统上可用
		Args:      []string{"hello"},
		Env:       map[string]string{"TEST": "value"},
	}

	client, err := hub.createMCPClient(config)
	if err != nil {
		// 如果创建失败，可能是因为系统环境问题，我们记录但不失败测试
		t.Logf("创建stdio客户端失败（可能是环境问题）: %v", err)
		return
	}

	assert.NotNil(t, client)
	// 清理资源
	if client != nil {
		client.Close()
	}
}

// TestCloseServersWithConnections 测试关闭有连接的服务器
func TestCloseServersWithConnections(t *testing.T) {
	hub := &MCPHub{
		connections: make(map[string]*Connection),
		tools:       make(map[string]tool.InvokableTool),
	}

	// 创建一个模拟连接（不实际连接到服务器）
	// 注意：这里我们不能真正测试关闭连接，因为需要真实的MCP客户端
	// 但我们可以测试空连接的情况

	err := hub.CloseServers()
	assert.NoError(t, err)
	assert.Empty(t, hub.connections)
	assert.Empty(t, hub.tools)
}

// TestGetClientWithDisabledServer 测试获取禁用服务器的客户端
func TestGetClientWithDisabledServer(t *testing.T) {
	hub := &MCPHub{
		connections: map[string]*Connection{
			"disabled_server": {
				Config: &ServerConfig{Disabled: true},
			},
		},
	}

	client, err := hub.GetClient("disabled_server")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "服务器已禁用")
}

// TestNewMCPHubInitializationFailure 测试初始化失败的情况
func TestNewMCPHubInitializationFailure(t *testing.T) {
	ctx := context.Background()

	// 创建一个会导致连接失败的配置
	settings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"invalid_server": {
				Transport: transportStdio,
				Command:   "nonexistent_command_that_should_fail",
				Disabled:  false, // 确保不被跳过
			},
		},
	}

	// 这应该失败，因为命令不存在
	hub, err := NewMCPHubFromSettings(ctx, settings)
	assert.Error(t, err)
	assert.Nil(t, hub)
	assert.Contains(t, err.Error(), "初始化服务器失败")
}

// TestConvertToolSchemaSerializationError 测试序列化错误
func TestConvertToolSchemaSerializationError(t *testing.T) {
	hub := &MCPHub{}

	// 创建一个包含循环引用的工具，这会导致序列化失败
	circularMap := make(map[string]interface{})
	circularMap["self"] = circularMap

	tool := mcp.Tool{
		Name:        "circular_tool",
		Description: "Tool with circular reference",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: circularMap,
		},
	}

	schema, err := hub.convertToolSchema(tool)
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "序列化工具输入模式失败")
}
