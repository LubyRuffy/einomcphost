// package einomcphost provides MCP (Model Context Protocol) server management functionality.
package einomcphost

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// ConnectionPool 用于管理MCP服务器连接池，确保每个服务器只连接一次
type ConnectionPool struct {
	mu          sync.RWMutex
	hubPool     map[string]*MCPHub   // 根据配置哈希存储的MCPHub池
	refCounts   map[string]int       // 引用计数器
	lastAccess  map[string]time.Time // 最后访问时间
	cleanupDone chan struct{}        // 清理goroutine完成信号
	isCleaning  bool                 // 是否正在执行清理
	maxIdleTime time.Duration        // 最大空闲时间
	serverHub   map[string]string    // 服务器名称到配置键的映射，用于快速查找
}

var (
	// 全局连接池单例
	globalPool     *ConnectionPool
	globalPoolOnce sync.Once
)

// GetConnectionPool 返回全局连接池单例
func GetConnectionPool() *ConnectionPool {
	globalPoolOnce.Do(func() {
		globalPool = NewConnectionPool()
	})
	return globalPool
}

// NewConnectionPool 创建一个新的连接池
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		hubPool:     make(map[string]*MCPHub),
		refCounts:   make(map[string]int),
		lastAccess:  make(map[string]time.Time),
		cleanupDone: make(chan struct{}),
		maxIdleTime: 30 * time.Minute,        // 默认30分钟无访问则清理
		serverHub:   make(map[string]string), // 初始化服务器名称到配置键的映射
	}

	// 启动清理协程
	go pool.startCleanupWorker()

	return pool
}

// 生成配置哈希键
func generateConfigKey(settings *MCPSettings) string {
	// 基于服务器配置内容生成唯一键
	if settings == nil || len(settings.MCPServers) == 0 {
		return "empty_config"
	}

	var keyBuilder strings.Builder

	// 构建包含所有服务器信息的键
	for name, config := range settings.MCPServers {
		if config.Disabled {
			continue // 忽略已禁用的服务器
		}

		// 根据传输类型构建不同的键
		keyBuilder.WriteString(fmt.Sprintf("|%s:", name))

		switch config.Transport {
		case transportSSE:
			keyBuilder.WriteString(fmt.Sprintf("SSE:%s", config.URL))
		case transportHTTP1, transportHTTPStreamable:
			keyBuilder.WriteString(fmt.Sprintf("HTTP:%s", config.URL))
		case transportStdio, "":
			keyBuilder.WriteString(fmt.Sprintf("STDIO:%s:", config.Command))
			// 添加参数
			for i, arg := range config.Args {
				if i > 0 {
					keyBuilder.WriteString(",")
				}
				keyBuilder.WriteString(arg)
			}
		}
	}

	key := keyBuilder.String()
	if key == "" {
		return "no_active_servers"
	}

	return key
}

// 检查连接是否健康
func (p *ConnectionPool) checkConnectionHealth(hub *MCPHub, configKey string) bool {
	// 简单检查是否有连接可用
	if len(hub.connections) == 0 {
		return false
	}

	// 如果有连接可用，尝试获取工具列表以验证连接是否正常
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 尝试调用GetToolsMap方法，这个方法会检查所有服务器
	_, err := hub.GetToolsMap(ctx)
	if err != nil {
		log.Printf("连接池中的连接不健康: %s, 错误: %v", configKey, err)
		return false
	}

	return true
}

// GetHub 获取或创建一个MCPHub实例
func (p *ConnectionPool) GetHub(ctx context.Context, settings *MCPSettings) (*MCPHub, error) {
	configKey := generateConfigKey(settings)

	// 尝试从池中获取现有连接
	p.mu.RLock()
	hub, exists := p.hubPool[configKey]
	p.mu.RUnlock()

	if exists {
		// 检查连接是否健康
		if !p.checkConnectionHealth(hub, configKey) {
			log.Printf("连接池中的连接不健康，强制关闭并重新连接")
			// 强制关闭不健康的连接
			p.ForceCloseHub(settings)
			// 继续执行后面的代码创建新连接
		} else {
			// 连接健康，更新引用计数和最后访问时间
			p.mu.Lock()
			p.refCounts[configKey]++
			p.lastAccess[configKey] = time.Now()

			// 更新服务器名称到配置键的映射
			for name := range settings.MCPServers {
				if !settings.MCPServers[name].Disabled {
					p.serverHub[name] = configKey
				}
			}

			p.mu.Unlock()

			log.Printf("复用已有MCP服务器连接池，当前引用计数: %d", p.refCounts[configKey])
			return hub, nil
		}
	}

	// 创建新连接
	newHub, err := NewMCPHubFromSettings(ctx, settings)
	if err != nil {
		return nil, fmt.Errorf("创建MCP服务器连接失败: %w", err)
	}

	// 添加到连接池
	p.mu.Lock()
	p.hubPool[configKey] = newHub
	p.refCounts[configKey] = 1
	p.lastAccess[configKey] = time.Now()

	// 更新服务器名称到配置键的映射
	for name := range settings.MCPServers {
		if !settings.MCPServers[name].Disabled {
			p.serverHub[name] = configKey
		}
	}

	p.mu.Unlock()

	log.Printf("创建新的MCP服务器连接池")
	return newHub, nil
}

// GetHubByServerName 根据服务器名称获取已有的MCPHub实例，如果不存在则返回nil和错误
func (p *ConnectionPool) GetHubByServerName(serverName string) (*MCPHub, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	configKey, exists := p.serverHub[serverName]
	if !exists {
		return nil, fmt.Errorf("服务器 %s 未连接", serverName)
	}

	hub, exists := p.hubPool[configKey]
	if !exists {
		return nil, fmt.Errorf("服务器 %s 的连接不存在", serverName)
	}

	// 更新最后访问时间
	p.lastAccess[configKey] = time.Now()

	return hub, nil
}

// ReleaseHub 释放MCPHub实例的引用
func (p *ConnectionPool) ReleaseHub(settings *MCPSettings) {
	configKey := generateConfigKey(settings)

	p.mu.Lock()
	defer p.mu.Unlock()

	if count, exists := p.refCounts[configKey]; exists {
		if count > 1 {
			// 减少引用计数
			p.refCounts[configKey]--
			log.Printf("释放MCP服务器连接引用，剩余引用计数: %d", p.refCounts[configKey])
		} else {
			// 如果引用计数归零，更新最后访问时间，但不立即关闭
			// 由清理协程负责关闭长时间无人使用的连接
			p.refCounts[configKey] = 0
			p.lastAccess[configKey] = time.Now()
			log.Printf("MCP服务器连接引用计数归零，等待清理")
		}
	}
}

// ForceCloseHub 强制关闭一个MCPHub连接，不管引用计数
func (p *ConnectionPool) ForceCloseHub(settings *MCPSettings) error {
	configKey := generateConfigKey(settings)

	p.mu.Lock()
	defer p.mu.Unlock()

	hub, exists := p.hubPool[configKey]
	if !exists {
		return nil // 连接不存在，无需关闭
	}

	// 关闭连接
	err := hub.CloseServers()

	// 无论成功失败，都从池中删除
	delete(p.hubPool, configKey)
	delete(p.refCounts, configKey)
	delete(p.lastAccess, configKey)

	// 清理服务器名称到配置键的映射
	for name, key := range p.serverHub {
		if key == configKey {
			delete(p.serverHub, name)
		}
	}

	log.Printf("强制关闭MCP服务器连接")
	return err
}

// CloseAllHubs 关闭所有连接
func (p *ConnectionPool) CloseAllHubs() []error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errors []error

	// 关闭所有连接
	for key, hub := range p.hubPool {
		if err := hub.CloseServers(); err != nil {
			errors = append(errors, fmt.Errorf("关闭连接 %s 失败: %w", key, err))
		}
	}

	// 清空映射
	p.hubPool = make(map[string]*MCPHub)
	p.refCounts = make(map[string]int)
	p.lastAccess = make(map[string]time.Time)
	p.serverHub = make(map[string]string)

	log.Printf("关闭所有MCP服务器连接")
	return errors
}

// Shutdown 关闭连接池，停止清理协程
func (p *ConnectionPool) Shutdown() []error {
	// 通知清理协程退出
	p.mu.Lock()
	if p.isCleaning {
		p.isCleaning = false
		close(p.cleanupDone)
	}
	p.mu.Unlock()

	// 关闭所有连接
	return p.CloseAllHubs()
}

// startCleanupWorker 启动清理工作协程
func (p *ConnectionPool) startCleanupWorker() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟检查一次
	defer ticker.Stop()

	p.mu.Lock()
	p.isCleaning = true
	p.mu.Unlock()

	for {
		select {
		case <-ticker.C:
			p.cleanupIdleConnections()
		case <-p.cleanupDone:
			return
		}
	}
}

// cleanupIdleConnections 清理空闲连接
func (p *ConnectionPool) cleanupIdleConnections() {
	now := time.Now()

	p.mu.Lock()
	defer p.mu.Unlock()

	for key, lastAccess := range p.lastAccess {
		// 检查引用计数和空闲时间
		if count, exists := p.refCounts[key]; exists && count == 0 {
			if now.Sub(lastAccess) > p.maxIdleTime {
				// 空闲时间超过阈值，关闭连接
				hub := p.hubPool[key]
				if hub != nil {
					if err := hub.CloseServers(); err != nil {
						log.Printf("清理空闲连接 %s 失败: %v", key, err)
					}
				}

				// 从池中删除
				delete(p.hubPool, key)
				delete(p.refCounts, key)
				delete(p.lastAccess, key)

				// 清理服务器名称到配置键的映射
				for name, configKey := range p.serverHub {
					if configKey == key {
						delete(p.serverHub, name)
					}
				}

				log.Printf("清理长时间空闲的MCP服务器连接")
			}
		}
	}
}
