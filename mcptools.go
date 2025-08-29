// Package einomcphost provides a MCP tools collection for Eino.
// It allows you to get MCP tools from a MCP server and use them in Eino.
// It also provides a way to close the MCP tools collection.
// 上层业务图省事可以用这个最简单，但是切记只适合只调用一次的场景，因为每调用一次就会全部连接一次mcp servers。
package einomcphost

import (
	"context"
	"errors"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/server"
)

// McpToolFnOptions 获取MCP工具的选项
type McpToolFnOptions struct {
	clientMcpServer *server.MCPServer // 内嵌的客户端MCP服务器
}

func WithClientMcpServer(clientMcpServer *server.MCPServer) McpToolOptionFn {
	return func(options *McpToolFnOptions) {
		options.clientMcpServer = clientMcpServer
	}
}

// McpToolOptionFn 获取MCP工具的选项函数
type McpToolOptionFn func(*McpToolFnOptions)

// MCPToolsCollection MCP工具集合
type MCPToolsCollection struct {
	tools      []tool.BaseTool          // 工具列表
	toolsMap   map[string]tool.BaseTool // 工具名称到工具的映射
	deferFuncs []func() error           // 延迟关闭函数列表
}

// GetAllTools 获取所有工具
func (mcpToolsCollection *MCPToolsCollection) GetAllTools() []tool.BaseTool {
	return mcpToolsCollection.tools
}

// GetTools 获取工具列表
func (mcpToolsCollection *MCPToolsCollection) GetTools(names []string) []tool.BaseTool {
	if mcpToolsCollection.toolsMap == nil {
		return nil
	}

	tools := []tool.BaseTool{}
	for _, name := range names {
		if tool, ok := mcpToolsCollection.toolsMap[name]; ok {
			tools = append(tools, tool)
		} else {
			log.Printf("[WARNING] tool %s not found", name)
		}
	}
	return tools
}

// Close 关闭MCP工具集合
func (mcpToolsCollection *MCPToolsCollection) Close() error {
	errs := []error{}
	for _, f := range mcpToolsCollection.deferFuncs {
		err := f()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// GetMcpTools 获取MCP工具
// 这个函数在一个进程内只适合调用一次，因为NewMCPHub后的服务器隐藏到函数中，没法共享和修改
// configPath 配置文件路径
// toolNames 工具名称列表
// options 选项
// 返回值:
// - 工具列表
// - 错误
func GetMcpTools(ctx context.Context, configPath string, toolNames []string, options ...McpToolOptionFn) (*MCPToolsCollection, error) {
	mcpToolsCollection := &MCPToolsCollection{
		toolsMap: make(map[string]tool.BaseTool),
	}

	mcpToolFnOptions := &McpToolFnOptions{}
	for _, option := range options {
		option(mcpToolFnOptions)
	}

	hubOptions := []MCPHubOption{}
	if mcpToolFnOptions.clientMcpServer != nil {
		inProcessClient, err := client.NewInProcessClient(mcpToolFnOptions.clientMcpServer)
		if err != nil {
			log.Fatal(err)
		}
		// defer inProcessClient.Close()
		mcpToolsCollection.deferFuncs = append(mcpToolsCollection.deferFuncs, inProcessClient.Close)
		hubOptions = append(hubOptions, WithInprocessMCPClient("inprocess", inProcessClient))
	}

	// 创建MCPHub
	hub, err := NewMCPHub(ctx, configPath, hubOptions...)
	if err != nil {
		log.Fatal(err)
	}
	// defer hub.CloseServers()
	mcpToolsCollection.deferFuncs = append(mcpToolsCollection.deferFuncs, hub.CloseServers)

	// 获取工具map
	toolsMap, err := hub.GetToolsMap(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(toolsMap)

	// 获取工具
	mcpToolsCollection.tools, err = hub.GetEinoTools(ctx, toolNames)
	if err != nil {
		return nil, err
	}
	// 构造tools map
	for _, tool := range mcpToolsCollection.tools {
		toolInfo, err := tool.Info(ctx)
		if err != nil {
			log.Fatal(err)
		}
		mcpToolsCollection.toolsMap[toolInfo.Name] = tool
	}

	return mcpToolsCollection, nil
}
