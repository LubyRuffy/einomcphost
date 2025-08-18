package einomcphost

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E测试需要使用build tag来区分，避免在普通测试时运行
// 运行方式: go test -tags=e2e -v ./...

const (
	testServerBinary = "test_mcp_server"
	testTimeout      = 60 * time.Second // 增加超时时间
)

// TestE2EStdioTransport 测试stdio transport的端到端功能
func TestE2EStdioTransport(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	// 创建配置
	config := fmt.Sprintf(`{
		"mcpServers": {
			"test_stdio_server": {
				"transport": "stdio",
				"command": "%s",
				"args": ["stdio"],
				"timeout": 30000000000
			}
		}
	}`, serverPath)

	// 创建MCPHub
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	hub, err := NewMCPHubFromString(ctx, config)
	require.NoError(t, err, "Failed to create MCP hub for stdio transport")
	defer hub.CloseServers()

	// 测试获取工具
	tools, err := hub.GetEinoTools(ctx, nil)
	require.NoError(t, err, "Failed to get tools from stdio server")
	assert.GreaterOrEqual(t, len(tools), 3, "Should have at least 3 tools (sum, multiply, echo)")

	// 验证特定工具存在
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		require.NoError(t, err)
		toolNames[info.Name] = true
		t.Logf("Found tool: %s", info.Name) // 打印工具名称以便调试
	}

	assert.True(t, toolNames["sum"], "Should have sum tool")
	assert.True(t, toolNames["multiply"], "Should have multiply tool")
	assert.True(t, toolNames["echo"], "Should have echo tool")

	// 测试工具调用（使用工具key，即serverName_toolName）
	testToolInvocation(t, ctx, hub, "test_stdio_server_sum", map[string]interface{}{
		"a": 5,
		"b": 3,
	}, "8")

	testToolInvocation(t, ctx, hub, "test_stdio_server_multiply", map[string]interface{}{
		"a": 4,
		"b": 6,
	}, "24")

	testToolInvocation(t, ctx, hub, "test_stdio_server_echo", map[string]interface{}{
		"message": "Hello E2E Test",
	}, "Echo: Hello E2E Test")
}

// TestE2EHTTPTransport 测试HTTP transport的端到端功能
func TestE2EHTTPTransport(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	// 找到可用端口
	port := findAvailablePort(t)

	// 启动HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	serverCmd := exec.CommandContext(ctx, serverPath, "http")
	serverCmd.Env = append(os.Environ(), fmt.Sprintf("MCP_SERVER_PORT=%d", port))

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start HTTP server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
	}()

	// 等待服务器启动
	waitForServer(t, port)

	// 创建配置
	config := fmt.Sprintf(`{
		"mcpServers": {
			"test_http_server": {
				"transport": "http",
				"url": "http://localhost:%d/mcp",
				"timeout": 60000000000
			}
		}
	}`, port)

	// 创建MCPHub
	hub, err := NewMCPHubFromString(ctx, config)
	require.NoError(t, err, "Failed to create MCP hub for HTTP transport")
	defer hub.CloseServers()

	// 测试获取工具
	tools, err := hub.GetEinoTools(ctx, nil)
	require.NoError(t, err, "Failed to get tools from HTTP server")
	assert.GreaterOrEqual(t, len(tools), 3, "Should have at least 3 tools")

	// 验证工具并测试调用
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		require.NoError(t, err)
		toolNames[info.Name] = true
	}

	assert.True(t, toolNames["sum"], "Should have sum tool")

	// 测试工具调用
	testToolInvocation(t, ctx, hub, "test_http_server_sum", map[string]interface{}{
		"a": 7,
		"b": 2,
	}, "9")
}

// TestE2EStreamableTransport 测试streamable transport的端到端功能
func TestE2EStreamableTransport(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	// 找到可用端口
	port := findAvailablePort(t)

	// 启动streamable服务器
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	serverCmd := exec.CommandContext(ctx, serverPath, "streamable")
	serverCmd.Env = append(os.Environ(), fmt.Sprintf("MCP_SERVER_PORT=%d", port))

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start streamable server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
	}()

	// 等待服务器启动
	waitForServer(t, port)

	// 创建配置
	config := fmt.Sprintf(`{
		"mcpServers": {
			"test_streamable_server": {
				"transport": "streamable",
				"url": "http://localhost:%d/mcp",
				"timeout": 30000000000
			}
		}
	}`, port)

	// 创建MCPHub
	hub, err := NewMCPHubFromString(ctx, config)
	require.NoError(t, err, "Failed to create MCP hub for streamable transport")
	defer hub.CloseServers()

	// 测试获取工具
	tools, err := hub.GetEinoTools(ctx, nil)
	require.NoError(t, err, "Failed to get tools from streamable server")
	assert.GreaterOrEqual(t, len(tools), 3, "Should have at least 3 tools")

	// 测试工具调用
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		require.NoError(t, err)
		toolNames[info.Name] = true
	}

	assert.True(t, toolNames["multiply"], "Should have multiply tool")

	// 测试工具调用
	testToolInvocation(t, ctx, hub, "test_streamable_server_multiply", map[string]interface{}{
		"a": 3,
		"b": 4,
	}, "12")
}

// TestE2ESSETransport 测试SSE transport的端到端功能
func TestE2ESSETransport(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	// 找到可用端口
	port := findAvailablePort(t)

	// 启动SSE服务器
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	serverCmd := exec.CommandContext(ctx, serverPath, "sse")
	serverCmd.Env = append(os.Environ(), fmt.Sprintf("MCP_SERVER_PORT=%d", port))

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start SSE server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
	}()

	// 等待服务器启动
	waitForServer(t, port)

	// 创建配置
	config := fmt.Sprintf(`{
		"mcpServers": {
			"test_sse_server": {
				"transport": "sse",
				"url": "http://localhost:%d/sse",
				"timeout": 30000000000
			}
		}
	}`, port)

	// 创建MCPHub
	hub, err := NewMCPHubFromString(ctx, config)
	require.NoError(t, err, "Failed to create MCP hub for SSE transport")
	defer hub.CloseServers()

	// 测试获取工具
	tools, err := hub.GetEinoTools(ctx, nil)
	require.NoError(t, err, "Failed to get tools from SSE server")
	assert.GreaterOrEqual(t, len(tools), 3, "Should have at least 3 tools")

	// 测试工具调用
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		require.NoError(t, err)
		toolNames[info.Name] = true
	}

	assert.True(t, toolNames["echo"], "Should have echo tool")

	// 测试工具调用
	testToolInvocation(t, ctx, hub, "test_sse_server_echo", map[string]interface{}{
		"message": "SSE Test Message",
	}, "Echo: SSE Test Message")
}

// TestE2EAllTransportsIntegration 测试多个transport同时工作
func TestE2EAllTransportsIntegration(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	// 找到可用端口
	httpPort := findAvailablePort(t)
	streamablePort := findAvailablePort(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// 启动HTTP服务器
	httpCmd := exec.CommandContext(ctx, serverPath, "http")
	httpCmd.Env = append(os.Environ(), fmt.Sprintf("MCP_SERVER_PORT=%d", httpPort))
	if err := httpCmd.Start(); err != nil {
		t.Fatalf("Failed to start HTTP server: %v", err)
	}
	defer func() {
		if httpCmd.Process != nil {
			httpCmd.Process.Kill()
			httpCmd.Wait()
		}
	}()

	// 启动streamable服务器
	streamableCmd := exec.CommandContext(ctx, serverPath, "streamable")
	streamableCmd.Env = append(os.Environ(), fmt.Sprintf("MCP_SERVER_PORT=%d", streamablePort))
	if err := streamableCmd.Start(); err != nil {
		t.Fatalf("Failed to start streamable server: %v", err)
	}
	defer func() {
		if streamableCmd.Process != nil {
			streamableCmd.Process.Kill()
			streamableCmd.Wait()
		}
	}()

	// 等待服务器启动
	waitForServer(t, httpPort)
	waitForServer(t, streamablePort)

	// 创建包含多种transport的配置
	config := fmt.Sprintf(`{
		"mcpServers": {
			"stdio_server": {
				"transport": "stdio",
				"command": "%s",
				"args": ["stdio"],
				"timeout": 30000000000
			},
			"http_server": {
				"transport": "http",
				"url": "http://localhost:%d/mcp",
				"timeout": 30000000000
			},
			"streamable_server": {
				"transport": "streamable",
				"url": "http://localhost:%d/mcp",
				"timeout": 30000000000
			}
		}
	}`, serverPath, httpPort, streamablePort)

	// 创建MCPHub
	hub, err := NewMCPHubFromString(ctx, config)
	require.NoError(t, err, "Failed to create MCP hub with multiple transports")
	defer hub.CloseServers()

	// 测试获取所有工具
	tools, err := hub.GetEinoTools(ctx, nil)
	require.NoError(t, err, "Failed to get tools from all servers")

	// 应该有3个服务器 × 3个工具 = 9个工具
	assert.GreaterOrEqual(t, len(tools), 9, "Should have tools from all three servers")

	// 测试每种transport的工具调用
	testToolInvocation(t, ctx, hub, "stdio_server_sum", map[string]interface{}{
		"a": 10,
		"b": 5,
	}, "15")

	testToolInvocation(t, ctx, hub, "http_server_multiply", map[string]interface{}{
		"a": 6,
		"b": 7,
	}, "42")

	testToolInvocation(t, ctx, hub, "streamable_server_echo", map[string]interface{}{
		"message": "Integration Test",
	}, "Echo: Integration Test")
}

// 辅助函数

// buildTestServer 构建测试服务器
func buildTestServer(t *testing.T) string {
	t.Helper()

	// 创建临时目录
	tempDir := t.TempDir()
	serverPath := filepath.Join(tempDir, testServerBinary)

	// 构建服务器
	cmd := exec.Command("go", "build", "-o", serverPath, "./testdata/test_mcp_server.go")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build test server: %v\nOutput: %s", err, string(output))
	}

	return serverPath
}

// cleanupTestServer 清理测试服务器
func cleanupTestServer(serverPath string) {
	os.Remove(serverPath)
}

// findAvailablePort 找到可用的端口
func findAvailablePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port
}

// waitForServer 等待服务器启动
func waitForServer(t *testing.T, port int) {
	t.Helper()

	timeout := time.After(20 * time.Second)          // 增加超时时间
	ticker := time.NewTicker(200 * time.Millisecond) // 降低检查频率
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("Server on port %d did not start within timeout", port)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
			if err == nil {
				conn.Close()
				t.Logf("Server on port %d is ready", port)
				// 额外等待一点时间确保服务完全就绪
				time.Sleep(2000 * time.Millisecond)
				return
			}
			t.Logf("Waiting for server on port %d: %v", port, err)
		}
	}
}

// TestE2EAllowedTools 测试AllowedTools功能
func TestE2EAllowedTools(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("允许部分工具", func(t *testing.T) {
		// 只允许sum和echo工具
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test_allowed_server": {
					"transport": "stdio",
					"command": "%s",
					"args": ["stdio"],
					"timeout": 30000000000,
					"allowedTools": ["sum", "echo"]
				}
			}
		}`, serverPath)

		hub, err := NewMCPHubFromString(ctx, config)
		require.NoError(t, err, "Failed to create MCP hub with allowedTools")
		defer hub.CloseServers()

		// 获取工具
		tools, err := hub.GetEinoTools(ctx, nil)
		require.NoError(t, err, "Failed to get tools")

		// 验证只有允许的工具存在
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			info, err := tool.Info(ctx)
			require.NoError(t, err)
			toolNames[info.Name] = true
		}

		// 应该有sum和echo工具
		assert.True(t, toolNames["sum"], "Should have sum tool")
		assert.True(t, toolNames["echo"], "Should have echo tool")
		// 不应该有multiply工具
		assert.False(t, toolNames["multiply"], "Should not have multiply tool")

		// 总共应该只有2个工具
		assert.Equal(t, 2, len(tools), "Should have exactly 2 tools")

		// 测试允许的工具可以正常调用
		testToolInvocation(t, ctx, hub, "test_allowed_server_sum", map[string]interface{}{
			"a": 3,
			"b": 4,
		}, "7")

		testToolInvocation(t, ctx, hub, "test_allowed_server_echo", map[string]interface{}{
			"message": "AllowedTools Test",
		}, "Echo: AllowedTools Test")
	})

	t.Run("只允许单个工具", func(t *testing.T) {
		// 只允许sum工具
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test_single_allowed_server": {
					"transport": "stdio",
					"command": "%s",
					"args": ["stdio"],
					"timeout": 30000000000,
					"allowedTools": ["sum"]
				}
			}
		}`, serverPath)

		hub, err := NewMCPHubFromString(ctx, config)
		require.NoError(t, err, "Failed to create MCP hub with single allowedTool")
		defer hub.CloseServers()

		tools, err := hub.GetEinoTools(ctx, nil)
		require.NoError(t, err, "Failed to get tools")

		// 验证只有sum工具存在
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			info, err := tool.Info(ctx)
			require.NoError(t, err)
			toolNames[info.Name] = true
		}

		assert.True(t, toolNames["sum"], "Should have sum tool")
		assert.False(t, toolNames["multiply"], "Should not have multiply tool")
		assert.False(t, toolNames["echo"], "Should not have echo tool")
		assert.Equal(t, 1, len(tools), "Should have exactly 1 tool")

		// 测试工具调用
		testToolInvocation(t, ctx, hub, "test_single_allowed_server_sum", map[string]interface{}{
			"a": 5,
			"b": 2,
		}, "7")
	})
}

// TestE2EExcludedTools 测试ExcludedTools功能
func TestE2EExcludedTools(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("排除部分工具", func(t *testing.T) {
		// 排除multiply工具
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test_excluded_server": {
					"transport": "stdio",
					"command": "%s",
					"args": ["stdio"],
					"timeout": 30000000000,
					"excludedTools": ["multiply"]
				}
			}
		}`, serverPath)

		hub, err := NewMCPHubFromString(ctx, config)
		require.NoError(t, err, "Failed to create MCP hub with excludedTools")
		defer hub.CloseServers()

		tools, err := hub.GetEinoTools(ctx, nil)
		require.NoError(t, err, "Failed to get tools")

		// 验证排除的工具不存在
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			info, err := tool.Info(ctx)
			require.NoError(t, err)
			toolNames[info.Name] = true
		}

		// 应该有sum和echo工具
		assert.True(t, toolNames["sum"], "Should have sum tool")
		assert.True(t, toolNames["echo"], "Should have echo tool")
		// 不应该有multiply工具
		assert.False(t, toolNames["multiply"], "Should not have multiply tool")

		// 总共应该有2个工具
		assert.Equal(t, 2, len(tools), "Should have exactly 2 tools")

		// 测试可用工具的调用
		testToolInvocation(t, ctx, hub, "test_excluded_server_sum", map[string]interface{}{
			"a": 8,
			"b": 2,
		}, "10")

		testToolInvocation(t, ctx, hub, "test_excluded_server_echo", map[string]interface{}{
			"message": "ExcludedTools Test",
		}, "Echo: ExcludedTools Test")
	})

	t.Run("排除多个工具", func(t *testing.T) {
		// 排除sum和echo工具，只保留multiply
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test_multi_excluded_server": {
					"transport": "stdio",
					"command": "%s",
					"args": ["stdio"],
					"timeout": 30000000000,
					"excludedTools": ["sum", "echo"]
				}
			}
		}`, serverPath)

		hub, err := NewMCPHubFromString(ctx, config)
		require.NoError(t, err, "Failed to create MCP hub with multiple excludedTools")
		defer hub.CloseServers()

		tools, err := hub.GetEinoTools(ctx, nil)
		require.NoError(t, err, "Failed to get tools")

		// 验证只有multiply工具存在
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			info, err := tool.Info(ctx)
			require.NoError(t, err)
			toolNames[info.Name] = true
		}

		assert.False(t, toolNames["sum"], "Should not have sum tool")
		assert.False(t, toolNames["echo"], "Should not have echo tool")
		assert.True(t, toolNames["multiply"], "Should have multiply tool")
		assert.Equal(t, 1, len(tools), "Should have exactly 1 tool")

		// 测试工具调用
		testToolInvocation(t, ctx, hub, "test_multi_excluded_server_multiply", map[string]interface{}{
			"a": 3,
			"b": 4,
		}, "12")
	})
}

// TestE2EToolsFilteringCombination 测试AllowedTools和ExcludedTools组合使用
func TestE2EToolsFilteringCombination(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("AllowedTools优先级测试", func(t *testing.T) {
		// 配置allowedTools包含sum和multiply，但excludedTools排除multiply
		// 根据代码逻辑，应该先检查allowedTools，再检查excludedTools
		config := fmt.Sprintf(`{
			"mcpServers": {
				"test_combination_server": {
					"transport": "stdio",
					"command": "%s",
					"args": ["stdio"],
					"timeout": 30000000000,
					"allowedTools": ["sum", "multiply"],
					"excludedTools": ["multiply"]
				}
			}
		}`, serverPath)

		hub, err := NewMCPHubFromString(ctx, config)
		require.NoError(t, err, "Failed to create MCP hub with combined filtering")
		defer hub.CloseServers()

		tools, err := hub.GetEinoTools(ctx, nil)
		require.NoError(t, err, "Failed to get tools")

		// 验证工具过滤结果
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			info, err := tool.Info(ctx)
			require.NoError(t, err)
			toolNames[info.Name] = true
		}

		// 应该只有sum工具（在allowedTools中但不在excludedTools中）
		assert.True(t, toolNames["sum"], "Should have sum tool")
		assert.False(t, toolNames["multiply"], "Should not have multiply tool (excluded)")
		assert.False(t, toolNames["echo"], "Should not have echo tool (not in allowedTools)")
		assert.Equal(t, 1, len(tools), "Should have exactly 1 tool")

		// 测试工具调用
		testToolInvocation(t, ctx, hub, "test_combination_server_sum", map[string]interface{}{
			"a": 6,
			"b": 3,
		}, "9")
	})
}

// TestE2EToolsFilteringWithHTTPTransport 测试HTTP transport下的工具过滤
func TestE2EToolsFilteringWithHTTPTransport(t *testing.T) {
	// 构建测试服务器
	serverPath := buildTestServer(t)
	defer cleanupTestServer(serverPath)

	// 找到可用端口
	port := findAvailablePort(t)

	// 启动HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	serverCmd := exec.CommandContext(ctx, serverPath, "http")
	serverCmd.Env = append(os.Environ(), fmt.Sprintf("MCP_SERVER_PORT=%d", port))

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start HTTP server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
	}()

	// 等待服务器启动
	waitForServer(t, port)

	// 测试HTTP transport下的工具过滤
	config := fmt.Sprintf(`{
		"mcpServers": {
			"test_http_filtered_server": {
				"transport": "http",
				"url": "http://localhost:%d/mcp",
				"timeout": 60000000000,
				"allowedTools": ["echo"]
			}
		}
	}`, port)

	hub, err := NewMCPHubFromString(ctx, config)
	require.NoError(t, err, "Failed to create MCP hub with HTTP transport and filtering")
	defer hub.CloseServers()

	tools, err := hub.GetEinoTools(ctx, nil)
	require.NoError(t, err, "Failed to get tools from HTTP server")

	// 验证只有echo工具存在
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		info, err := tool.Info(ctx)
		require.NoError(t, err)
		toolNames[info.Name] = true
	}

	assert.True(t, toolNames["echo"], "Should have echo tool")
	assert.False(t, toolNames["sum"], "Should not have sum tool")
	assert.False(t, toolNames["multiply"], "Should not have multiply tool")
	assert.Equal(t, 1, len(tools), "Should have exactly 1 tool")

	// 测试工具调用
	testToolInvocation(t, ctx, hub, "test_http_filtered_server_echo", map[string]interface{}{
		"message": "HTTP Filtered Test",
	}, "Echo: HTTP Filtered Test")
}

// testToolInvocation 测试工具调用
func testToolInvocation(t *testing.T, ctx context.Context, hub *MCPHub, toolName string, params map[string]interface{}, expectedResult string) {
	t.Helper()

	result, err := hub.InvokeTool(ctx, toolName, params)
	require.NoError(t, err, "Failed to invoke tool %s", toolName)

	// 处理JSON序列化的结果，去掉引号
	actualResult := result
	if len(result) >= 2 && result[0] == '"' && result[len(result)-1] == '"' {
		actualResult = result[1 : len(result)-1]
	}

	assert.Equal(t, expectedResult, actualResult, "Tool %s returned unexpected result", toolName)
}
