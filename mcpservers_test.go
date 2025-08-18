package einomcphost

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockInvokableTool 是一个模拟的 tool.InvokableTool 对象
type MockInvokableTool struct {
	mock.Mock
	Name string
}

// Info 模拟 Info 方法
func (m *MockInvokableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.ToolInfo), args.Error(1)
}

// Run 模拟 Run 方法
func (m *MockInvokableTool) Run(ctx context.Context, params map[string]interface{}) (string, error) {
	args := m.Called(ctx, params)
	return args.String(0), args.Error(1)
}

// InvokableRun 模拟 InvokableRun 方法
func (m *MockInvokableTool) InvokableRun(ctx context.Context, paramsStr string, opts ...tool.Option) (string, error) {
	args := m.Called(ctx, paramsStr)
	return args.String(0), args.Error(1)
}

// 跳过实际连接测试，因为它需要真实的MCP服务器
func TestLoadSettings_Integration(t *testing.T) {
	// 这个测试只测试配置加载，不测试实际连接
	t.Skip("Skipping integration test that requires real MCP servers")
}

func TestLoadSettings(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "load_test_mcpservers.json")

	// 创建测试配置
	config := MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"server1": {
				Transport: "stdio",
				Command:   "test_command1",
				Args:      []string{"arg1", "arg2"},
				Env:       map[string]string{"ENV1": "value1"},
				Disabled:  true,
			},
			"server2": {
				Transport: "sse",
				URL:       "http://test-url.com",
				Disabled:  true,
			},
		},
	}

	// 将配置写入文件
	configBytes, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configBytes, 0644)
	require.NoError(t, err)

	// 测试加载配置
	loadedConfig, err := LoadSettings(configPath)
	require.NoError(t, err)

	// 验证配置内容
	assert.Equal(t, 2, len(loadedConfig.MCPServers))

	// 验证server1配置
	server1, exists := loadedConfig.MCPServers["server1"]
	assert.True(t, exists)
	assert.Equal(t, "stdio", server1.Transport)
	assert.Equal(t, "test_command1", server1.Command)
	assert.Equal(t, []string{"arg1", "arg2"}, server1.Args)
	assert.Equal(t, map[string]string{"ENV1": "value1"}, server1.Env)
	assert.True(t, server1.Disabled)

	// 验证server2配置
	server2, exists := loadedConfig.MCPServers["server2"]
	assert.True(t, exists)
	assert.Equal(t, "sse", server2.Transport)
	assert.Equal(t, "http://test-url.com", server2.URL)
	assert.True(t, server2.Disabled)
}

func TestLoadSettingsInvalid(t *testing.T) {
	// 测试加载不存在的配置文件
	_, err := LoadSettings("non_existent_config.json")
	assert.Error(t, err)

	// 创建无效的配置文件
	tempDir := t.TempDir()
	invalidConfigPath := filepath.Join(tempDir, "invalid_config.json")
	err = os.WriteFile(invalidConfigPath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	// 测试加载无效的配置文件
	_, err = LoadSettings(invalidConfigPath)
	assert.Error(t, err)
}

func TestLoadSettingsFromString(t *testing.T) {
	// 有效的配置字符串
	validConfig := `{
		"mcpServers": {
			"test_server": {
				"transportType": "stdio",
				"command": "echo",
				"args": ["hello"],
				"env": {"TEST_ENV": "test_value"}
			}
		}
	}`

	// 测试加载有效配置
	settings, err := LoadSettingsFromString(validConfig)
	require.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, 1, len(settings.MCPServers))

	// 无效的配置字符串
	invalidConfig := `{invalid json}`

	// 测试加载无效配置
	_, err = LoadSettingsFromString(invalidConfig)
	assert.Error(t, err)
}

func TestServerConfigGetTimeoutDuration(t *testing.T) {
	// 测试默认超时
	config := ServerConfig{}
	assert.Equal(t, time.Duration(DefaultMCPTimeoutSeconds)*time.Second, config.GetTimeoutDuration())

	// 测试自定义超时
	customTimeout := 60 * time.Second
	config.Timeout = customTimeout
	assert.Equal(t, customTimeout, config.GetTimeoutDuration())
}

func TestValidateSettings(t *testing.T) {
	// 测试有效配置
	validSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"stdio_server": {
				Transport: "stdio",
				Command:   "echo",
				Args:      []string{"hello"},
			},
			"sse_server": {
				Transport: "sse",
				URL:       "http://test-url.com",
			},
		},
	}

	err := validateSettings(validSettings)
	assert.NoError(t, err)

	// 测试无效配置 - 空设置
	err = validateSettings(nil)
	assert.Error(t, err)

	// 测试无效配置 - 超时太短
	invalidTimeoutSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"short_timeout": {
				Transport: "stdio",
				Command:   "echo",
				Timeout:   1 * time.Second, // 小于最小超时
			},
		},
	}

	err = validateSettings(invalidTimeoutSettings)
	assert.Error(t, err)

	// 测试无效配置 - SSE缺少URL
	invalidSSESettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"invalid_sse": {
				Transport: "sse",
				// 缺少URL
			},
		},
	}

	err = validateSettings(invalidSSESettings)
	assert.Error(t, err)

	// 测试无效配置 - stdio缺少命令
	invalidStdioSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"invalid_stdio": {
				Transport: "stdio",
				// 缺少Command
			},
		},
	}

	err = validateSettings(invalidStdioSettings)
	assert.Error(t, err)

	// 测试无效配置 - 不支持的传输类型
	invalidTransportSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"invalid_transport": {
				Transport: "invalid",
				Command:   "echo",
			},
		},
	}

	err = validateSettings(invalidTransportSettings)
	assert.Error(t, err)
}

// 测试MCPHub的GetEinoTools方法
func TestMCPHub_GetEinoTools(t *testing.T) {
	// 创建一个模拟的MCPHub
	hub := &MCPHub{
		tools: make(map[string]tool.InvokableTool),
	}

	// 创建一个模拟的工具
	mockTool := &MockInvokableTool{Name: "test_tool"}

	// 将模拟工具添加到hub
	hub.tools["server_test_tool"] = mockTool

	// 测试获取所有工具
	ctx := context.Background()
	tools, err := hub.GetEinoTools(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, tools, 1)

	// 测试按名称过滤工具
	tools, err = hub.GetEinoTools(ctx, []string{"server_test_tool"})
	assert.NoError(t, err)
	assert.Len(t, tools, 1)

	// 测试按名称过滤工具（不存在的工具）
	tools, err = hub.GetEinoTools(ctx, []string{"non_existent_tool"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "工具不存在")
}

// 测试MCPHub的InvokeTool方法
func TestMCPHub_InvokeTool(t *testing.T) {
	// 创建一个模拟的MCPHub
	hub := &MCPHub{
		tools: make(map[string]tool.InvokableTool),
	}

	// 创建一个模拟的工具
	mockTool := &MockInvokableTool{}
	mockTool.On("InvokableRun", mock.Anything, mock.Anything).Return("test result", nil)

	// 将模拟工具添加到hub
	hub.tools["server_test_tool"] = mockTool

	// 测试调用工具
	ctx := context.Background()
	result, err := hub.InvokeTool(ctx, "server_test_tool", map[string]interface{}{"param": "value"})
	assert.NoError(t, err)
	assert.Equal(t, "test result", result)

	// 测试调用不存在的工具
	_, err = hub.InvokeTool(ctx, "non_existent_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "工具不存在")

	// 验证方法调用
	mockTool.AssertExpectations(t)
}

// 测试MCPHub的CloseServers方法
func TestMCPHub_CloseServers(t *testing.T) {
	// 由于CloseServers方法依赖于外部组件，这里只测试基本功能
	hub := &MCPHub{
		connections: make(map[string]*Connection),
	}

	// 测试关闭空连接
	err := hub.CloseServers()
	assert.NoError(t, err)

	// 更复杂的测试需要模拟client.MCPClient接口，这超出了当前任务的范围
}
