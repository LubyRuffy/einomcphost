package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 检查启动模式
	mode := "stdio" // 默认stdio模式
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	// 创建MCP服务器
	s := server.NewMCPServer("test-mcp-server", "1.0.0")

	// 注册工具
	s.AddTool(
		mcp.NewTool("sum",
			mcp.WithDescription("sum two numbers"),
			mcp.WithNumber("a", mcp.DefaultNumber(1)),
			mcp.WithNumber("b", mcp.DefaultNumber(2)),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			a := request.GetArguments()["a"].(float64)
			b := request.GetArguments()["b"].(float64)
			result := a + b
			return mcp.NewToolResultText(fmt.Sprintf("%.0f", result)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("multiply",
			mcp.WithDescription("multiply two numbers"),
			mcp.WithNumber("a", mcp.DefaultNumber(1)),
			mcp.WithNumber("b", mcp.DefaultNumber(2)),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			a := request.GetArguments()["a"].(float64)
			b := request.GetArguments()["b"].(float64)
			result := a * b
			return mcp.NewToolResultText(fmt.Sprintf("%.0f", result)), nil
		},
	)

	s.AddTool(
		mcp.NewTool("echo",
			mcp.WithDescription("echo a message"),
			mcp.WithString("message", mcp.DefaultString("hello")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			message := request.GetArguments()["message"].(string)
			return mcp.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
		},
	)

	// 根据模式启动服务器
	switch mode {
	case "stdio":
		log.Println("Starting MCP server in stdio mode...")
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Failed to serve stdio: %v", err)
		}
		return // stdio模式是阻塞的，不需要等待信号
	case "sse", "http", "streamable":
		// HTTP/SSE/Streamable模式
		port := "28080"
		if p := os.Getenv("MCP_SERVER_PORT"); p != "" {
			port = p
		}

		log.Printf("Starting MCP server in %s mode on port %s...", mode, port)

		var srv interface {
			Start(addr string) error
		}
		if mode == "sse" {
			srv = server.NewSSEServer(s, server.WithSSEEndpoint("/sse"))
		} else { // http, streamable
			srv = server.NewStreamableHTTPServer(s, server.WithEndpointPath("/mcp"))
		}

		log.Printf("Starting server on port %s", port)
		if err := srv.Start(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}

		log.Printf("Server started: http://localhost:%s/mcp", port)
	default:
		log.Fatalf("Unknown mode: %s. Supported modes: stdio, sse, http, streamable", mode)
	}
}
