# einomcphost

`einomcphost` is a Go library that acts as a hub for multiple MCP (Model Context Protocol) servers. It simplifies the integration of various MCP-based tools into the Eino framework by discovering, managing, and exposing them through a unified interface.

## Features

* **Multiple Server Support**: Connect to and manage multiple MCP servers simultaneously.
* **Flexible Transports**: Supports both `stdio` and `sse` transport types for connecting to MCP servers.
* **Configuration Driven**: Load server configurations from a central JSON file (`mcpservers.json`).
* **Automatic Tool Discovery**: Automatically discovers available tools from all connected servers.
* **Eino Framework Integration**: Seamlessly converts MCP tool schemas and exposes them as standard `tool.InvokableTool` instances for use with Eino agents.
* **Connection Pooling**: Reuses existing connections to improve efficiency and reduce overhead.

## Configuration

Create a `mcpservers.json` file to define your MCP servers.

Here is an example configuration:
```json
{
  "mcpServers": {
    "fofa_mcp": {
      "transportType": "stdio",
      "command": "python",
      "args": [
        "-m",
        "fofa_mcp"
      ],
      "disabled": false
    },
    "another_sse_server": {
      "transportType": "sse",
      "url": "http://localhost:8080/events",
      "disabled": false
    }
  }
}
```

### ServerConfig Fields

*   `transportType`: (string) The transport mechanism. Can be `"stdio"` or `"sse"`. Defaults to `"stdio"`.
*   `command`: (string) Required for `stdio` transport. The command to execute.
*   `args`: ([]string) Optional arguments for the command.
*   `env`: (map[string]string) Optional environment variables for the command.
*   `url`: (string) Required for `sse` transport. The URL of the SSE server.
*   `disabled`: (bool) Set to `true` to disable the server.
*   `timeout`: (number) Operation timeout in seconds. Defaults to 30.

## Usage

Here's a basic example of how to use `einomcphost` to get tools and use them with an Eino agent.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/LubyRuffy/einomcphost"
    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/flow/agent"
    "github.com/cloudwego/eino/flow/agent/react"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 1. Create a new MCPHub from the configuration file
    hub, err := einomcphost.NewMCPHub(ctx, `mcpservers.json`)
    if err != nil {
        log.Fatal(err)
    }
    defer hub.CloseServers()

    // 2. Get the list of Eino tools you want to use
    // The tool names are prefixed with the server name from the config, e.g., "fofa_mcp_web_search"
    tools, err := hub.GetEinoTools(ctx, []string{"fofa_mcp_web_search"})
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Successfully loaded %d tools", len(tools))

    // 3. (Optional) You can also get a map of all available tools
    allTools, err := hub.GetToolsMap(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Available tools:")
    for toolName := range allTools {
        fmt.Println("- ", toolName)
    }

    // 4. Use the tools with an Eino agent
    // (This is a simplified example, see examples/tools/main.go for a complete implementation)
    /*
    openaiClient, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{ ... })
    reactAgent, _ := react.NewAgent(ctx, &react.AgentConfig{
        ToolCallingModel: openaiClient,
        ToolsConfig: compose.ToolsNodeConfig{
            Tools: tools,
        },
    })

    s, _ := reactAgent.Stream(ctx, []*schema.Message{
        {
            Role:    "user",
            Content: "Search for the founding year of Google",
        },
    })
    // ... process stream ...
    */
}
```

## 几个设计原则

* 默认从配置文件中加载 MCP 服务器配置，可以配置进程的方式加载 MCP 服务器配置（需要手动提供WithInprocessMCPClient）
* 不同的transport对应的路径不同，不能敷用：老版本的sse对应/sse的路径；新版本的http对应/mcp的路径
