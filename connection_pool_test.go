// package einomcphost provides MCP server management functionality.
package einomcphost

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnectionPool_GetHub(t *testing.T) {
	// 准备测试环境
	pool := NewConnectionPool()

	// 创建两个不同的配置
	settings1 := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"server1": {
				Command: "test1",
			},
		},
	}

	settings2 := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"server2": {
				Command: "test2",
			},
		},
	}

	// 模拟GetHub
	configKey1 := generateConfigKey(settings1)
	configKey2 := generateConfigKey(settings2)

	// 手动设置hubPool
	pool.mu.Lock()
	pool.hubPool[configKey1] = &MCPHub{}
	pool.refCounts[configKey1] = 1
	pool.lastAccess[configKey1] = time.Now()
	pool.mu.Unlock()

	// 验证相同配置返回相同的Hub
	pool.mu.RLock()
	hub1 := pool.hubPool[configKey1]
	pool.mu.RUnlock()

	assert.NotNil(t, hub1, "应该能获取到已存在的Hub")

	// 验证不同配置的键不同
	assert.NotEqual(t, configKey1, configKey2, "不同配置应该生成不同的键")

	// 清理
	pool.CloseAllHubs()
}

func TestConnectionPool_ReleaseHub(t *testing.T) {
	// 准备测试环境
	pool := NewConnectionPool()

	// 创建配置
	settings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"server1": {
				Command: "test",
			},
		},
	}

	// 手动设置hubPool和计数
	configKey := generateConfigKey(settings)

	pool.mu.Lock()
	pool.hubPool[configKey] = &MCPHub{}
	pool.refCounts[configKey] = 2
	pool.lastAccess[configKey] = time.Now()
	pool.mu.Unlock()

	// 释放Hub
	pool.ReleaseHub(settings)

	// 检查引用计数是否减少
	pool.mu.RLock()
	count := pool.refCounts[configKey]
	pool.mu.RUnlock()

	assert.Equal(t, 1, count, "引用计数应该减少到1")

	// 再次释放
	pool.ReleaseHub(settings)

	// 检查引用计数是否为0
	pool.mu.RLock()
	count = pool.refCounts[configKey]
	pool.mu.RUnlock()

	assert.Equal(t, 0, count, "引用计数应该减少到0")

	// 清理
	pool.CloseAllHubs()
}

func TestConnectionPool_Concurrency(t *testing.T) {
	// 准备测试环境
	pool := NewConnectionPool()

	// 创建配置
	settings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"server1": {
				Command: "test",
			},
		},
	}

	// 手动设置hubPool和计数
	configKey := generateConfigKey(settings)

	pool.mu.Lock()
	pool.hubPool[configKey] = &MCPHub{}
	pool.refCounts[configKey] = 0
	pool.lastAccess[configKey] = time.Now()
	pool.mu.Unlock()

	// 并发增加和释放Hub引用
	var wg sync.WaitGroup
	workers := 10

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 模拟获取Hub
			pool.mu.Lock()
			pool.refCounts[configKey]++
			pool.mu.Unlock()

			// 模拟使用Hub的工作
			time.Sleep(10 * time.Millisecond)

			// 释放Hub
			pool.ReleaseHub(settings)
		}()
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 检查引用计数是否为0
	pool.mu.RLock()
	count := pool.refCounts[configKey]
	pool.mu.RUnlock()

	assert.Equal(t, 0, count, "所有引用释放后计数应该为0")

	// 清理
	pool.CloseAllHubs()
}

func TestConnectionPool_CleanupIdleConnections(t *testing.T) {
	// 创建带有较短空闲时间的连接池
	pool := &ConnectionPool{
		hubPool:     make(map[string]*MCPHub),
		refCounts:   make(map[string]int),
		lastAccess:  make(map[string]time.Time),
		cleanupDone: make(chan struct{}),
		maxIdleTime: 100 * time.Millisecond, // 设置较短的空闲时间用于测试
	}

	// 创建配置
	settings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"server1": {
				Command: "test",
			},
		},
	}

	// 模拟一个已经存在的连接
	configKey := generateConfigKey(settings)
	pool.mu.Lock()
	pool.hubPool[configKey] = &MCPHub{}
	pool.refCounts[configKey] = 0
	pool.lastAccess[configKey] = time.Now().Add(-200 * time.Millisecond) // 设置为已经超过空闲时间
	pool.mu.Unlock()

	// 手动触发清理
	pool.cleanupIdleConnections()

	// 检查连接是否被清理
	pool.mu.RLock()
	_, exists := pool.hubPool[configKey]
	pool.mu.RUnlock()

	assert.False(t, exists, "空闲连接应该被清理")
}

func TestGlobalConnectionPool(t *testing.T) {
	// 获取全局连接池
	pool1 := GetConnectionPool()
	assert.NotNil(t, pool1)

	// 再次获取，应该是同一个实例
	pool2 := GetConnectionPool()
	assert.Equal(t, pool1, pool2, "应该返回相同的全局连接池实例")
}

// TestGenerateConfigKeyWithStreamableTransport 测试streamable transport的配置键生成
func TestGenerateConfigKeyWithStreamableTransport(t *testing.T) {
	tests := []struct {
		name     string
		settings *MCPSettings
		expected string
	}{
		{
			name: "streamable transport config",
			settings: &MCPSettings{
				MCPServers: map[string]*ServerConfig{
					"streamable_server": {
						Transport: transportHTTPStreamable,
						URL:       "http://localhost:8080/stream",
					},
				},
			},
			expected: "|streamable_server:HTTP:http://localhost:8080/stream",
		},
		{
			name: "HTTP transport config",
			settings: &MCPSettings{
				MCPServers: map[string]*ServerConfig{
					"http_server": {
						Transport: transportHTTP1,
						URL:       "http://localhost:8080/http",
					},
				},
			},
			expected: "|http_server:HTTP:http://localhost:8080/http",
		},
		{
			name: "mixed transports config",
			settings: &MCPSettings{
				MCPServers: map[string]*ServerConfig{
					"streamable_server": {
						Transport: transportHTTPStreamable,
						URL:       "http://localhost:8080/stream",
					},
					"stdio_server": {
						Transport: transportStdio,
						Command:   "python",
						Args:      []string{"-m", "server"},
					},
				},
			},
			// 注意：由于map的遍历顺序不确定，这里只检查包含关系
		},
		{
			name: "disabled streamable server should be ignored",
			settings: &MCPSettings{
				MCPServers: map[string]*ServerConfig{
					"disabled_server": {
						Transport: transportHTTPStreamable,
						URL:       "http://localhost:8080/stream",
						Disabled:  true,
					},
					"active_server": {
						Transport: transportStdio,
						Command:   "python",
					},
				},
			},
			expected: "|active_server:STDIO:python:",
		},
		{
			name: "empty config",
			settings: &MCPSettings{
				MCPServers: map[string]*ServerConfig{},
			},
			expected: "empty_config",
		},
		{
			name:     "nil settings",
			settings: nil,
			expected: "empty_config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := generateConfigKey(tt.settings)
			if tt.name == "mixed transports config" {
				// 对于混合配置，检查是否包含两个服务器的配置
				assert.Contains(t, key, "streamable_server:HTTP:http://localhost:8080/stream")
				assert.Contains(t, key, "stdio_server:STDIO:python:")
			} else {
				assert.Equal(t, tt.expected, key)
			}
		})
	}
}

// TestConnectionPoolWithStreamableTransport 测试连接池处理streamable transport
func TestConnectionPoolWithStreamableTransport(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.CloseAllHubs()

	// 创建streamable transport配置
	streamableSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"streamable_server": {
				Transport: transportHTTPStreamable,
				URL:       "http://localhost:8080/stream",
				Disabled:  false, // 不禁用，这样才能生成正确的配置键
			},
		},
	}

	// 创建HTTP transport配置
	httpSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"http_server": {
				Transport: transportHTTP1,
				URL:       "http://localhost:8080/http",
				Disabled:  false, // 不禁用，这样才能生成正确的配置键
			},
		},
	}

	// 生成配置键
	streamableKey := generateConfigKey(streamableSettings)
	httpKey := generateConfigKey(httpSettings)

	// 验证不同transport生成不同的键
	assert.NotEqual(t, streamableKey, httpKey, "不同的transport类型应该生成不同的配置键")

	// 验证streamable transport的键格式
	assert.Contains(t, streamableKey, "HTTP:http://localhost:8080/stream", "streamable transport应该使用HTTP前缀")
	assert.Contains(t, httpKey, "HTTP:http://localhost:8080/http", "HTTP transport也应该使用HTTP前缀")

	// 测试服务器名称映射
	pool.mu.Lock()
	pool.serverHub["streamable_server"] = streamableKey
	pool.serverHub["http_server"] = httpKey
	pool.mu.Unlock()

	// 测试通过服务器名称查找配置键
	pool.mu.RLock()
	foundKey1, exists1 := pool.serverHub["streamable_server"]
	foundKey2, exists2 := pool.serverHub["http_server"]
	pool.mu.RUnlock()

	assert.True(t, exists1, "应该能找到streamable服务器的配置键")
	assert.True(t, exists2, "应该能找到HTTP服务器的配置键")
	assert.Equal(t, streamableKey, foundKey1, "找到的streamable配置键应该匹配")
	assert.Equal(t, httpKey, foundKey2, "找到的HTTP配置键应该匹配")
}
