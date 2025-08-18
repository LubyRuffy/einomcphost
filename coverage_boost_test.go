package einomcphost

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestBuildEnvironment tests the buildEnvironment method
func TestBuildEnvironment(t *testing.T) {
	hub := &MCPHub{}

	tests := []struct {
		name     string
		envMap   map[string]string
		expected []string
	}{
		{
			name:     "nil environment",
			envMap:   nil,
			expected: nil,
		},
		{
			name:     "empty environment",
			envMap:   map[string]string{},
			expected: []string{},
		},
		{
			name:     "single environment variable",
			envMap:   map[string]string{"KEY": "value"},
			expected: []string{"KEY=value"},
		},
		{
			name: "multiple environment variables",
			envMap: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hub.buildEnvironment(tt.envMap)
			if len(tt.expected) == 0 && len(result) == 0 {
				return // Both are empty, test passes
			}
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

// TestCreateMCPClient tests the createMCPClient method with unsupported transport
func TestCreateMCPClient(t *testing.T) {
	hub := &MCPHub{}

	// Test unsupported transport type
	config := &ServerConfig{
		Transport: "unsupported",
	}

	client, err := hub.createMCPClient(config)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "不支持的传输类型")
}

// TestConvertToolSchema tests the convertToolSchema method
func TestConvertToolSchema(t *testing.T) {
	hub := &MCPHub{}

	// Test with a simple tool schema
	tool := mcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "A test parameter",
				},
			},
		},
	}

	schema, err := hub.convertToolSchema(tool)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
}

// TestConvertToolSchemaWithExclusiveFields tests convertToolSchema with exclusive fields
func TestConvertToolSchemaWithExclusiveFields(t *testing.T) {
	hub := &MCPHub{}

	// Test with exclusive fields that should be removed
	tool := mcp.Tool{
		Name:        "test_tool_exclusive",
		Description: "A test tool with exclusive fields",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"param1": map[string]interface{}{
					"type":             "number",
					"exclusiveMaximum": 100,
					"exclusiveMinimum": 0,
				},
			},
		},
	}

	schema, err := hub.convertToolSchema(tool)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
}

// TestNewMCPHubFromStringBasic tests NewMCPHubFromString with basic cases
func TestNewMCPHubFromStringBasic(t *testing.T) {
	ctx := context.Background()

	// Test with empty configuration
	emptyConfig := `{"mcpServers": {}}`
	hub, err := NewMCPHubFromString(ctx, emptyConfig)
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	hub.CloseServers()

	// Test with invalid JSON
	invalidConfig := `{"invalid": json}`
	hub, err = NewMCPHubFromString(ctx, invalidConfig)
	assert.Error(t, err)
	assert.Nil(t, hub)
}

// TestNewMCPHubFromSettingsBasic tests NewMCPHubFromSettings with basic cases
func TestNewMCPHubFromSettingsBasic(t *testing.T) {
	ctx := context.Background()

	// Test with empty settings
	settings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{},
	}
	hub, err := NewMCPHubFromSettings(ctx, settings)
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	hub.CloseServers()

	// Test with disabled server
	settings = &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"disabled": {
				Transport: transportStdio,
				Command:   "echo",
				Disabled:  true,
			},
		},
	}
	hub, err = NewMCPHubFromSettings(ctx, settings)
	assert.NoError(t, err)
	assert.NotNil(t, hub)
	hub.CloseServers()
}

// TestMCPHubGetClient tests the GetClient method
func TestMCPHubGetClient(t *testing.T) {
	hub := &MCPHub{
		connections: make(map[string]*Connection),
	}

	// Test getting non-existent client
	client, err := hub.GetClient("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "未找到服务器连接")
}

// TestCloseExistingConnection tests the closeExistingConnection method
func TestCloseExistingConnection(t *testing.T) {
	hub := &MCPHub{
		connections: make(map[string]*Connection),
	}

	// Test closing non-existent connection
	err := hub.closeExistingConnection("nonexistent")
	assert.NoError(t, err)
}

// TestInvokeToolNonExistent tests InvokeTool with non-existent tool
func TestInvokeToolNonExistent(t *testing.T) {
	hub := &MCPHub{
		tools: make(map[string]tool.InvokableTool),
	}

	result, err := hub.InvokeTool(context.Background(), "nonexistent", nil)
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "工具不存在")
}

// MockInvokableToolSimple is a simple mock for InvokableTool
type MockInvokableToolSimple struct {
	mock.Mock
}

func (m *MockInvokableToolSimple) Info(ctx context.Context) (*schema.ToolInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).(*schema.ToolInfo), args.Error(1)
}

func (m *MockInvokableToolSimple) Run(ctx context.Context, params map[string]interface{}) (string, error) {
	args := m.Called(ctx, params)
	return args.String(0), args.Error(1)
}

func (m *MockInvokableToolSimple) InvokableRun(ctx context.Context, paramsStr string, opts ...tool.Option) (string, error) {
	args := m.Called(ctx, paramsStr)
	return args.String(0), args.Error(1)
}

// TestInvokeToolSerializationError tests InvokeTool with serialization error
func TestInvokeToolSerializationError(t *testing.T) {
	hub := &MCPHub{
		tools: make(map[string]tool.InvokableTool),
	}

	mockTool := &MockInvokableToolSimple{}
	hub.tools["test_tool"] = mockTool

	// Create a map with circular reference to cause JSON serialization error
	circularMap := make(map[string]interface{})
	circularMap["self"] = circularMap

	result, err := hub.InvokeTool(context.Background(), "test_tool", circularMap)
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "序列化参数失败")
}

// TestGetEinoToolsEmpty tests GetEinoTools with empty hub
func TestGetEinoToolsEmpty(t *testing.T) {
	hub := &MCPHub{
		tools: make(map[string]tool.InvokableTool),
	}

	// Test getting all tools from empty hub
	tools, err := hub.GetEinoTools(context.Background(), nil)
	assert.NoError(t, err)
	assert.Empty(t, tools)

	// Test getting specific tool from empty hub
	tools, err = hub.GetEinoTools(context.Background(), []string{"nonexistent"})
	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "工具不存在")
}
