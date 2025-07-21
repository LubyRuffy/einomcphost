package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/LubyRuffy/einomcphost"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	template_callbacks "github.com/cloudwego/eino/utils/callbacks"
)

// buildAgentCallback 构建agent回调
func buildAgentCallback(debug bool) callbacks.Handler {
	modelHandler := &template_callbacks.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			log.Printf("model start: %s", info.Name)
			if debug {
				log.Printf("model start: %s, messages: %v", info.Name, input.Messages)
			}
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			log.Printf("model end: %s", info.Name)
			if len(output.Message.ToolCalls) > 0 {
				if debug {
					log.Printf("response nedd to do tool calls: %s, %v\n", info.Name, output.Message.ToolCalls)
				} else {
					log.Printf("response nedd to do tool calls: %s\n", info.Name)
				}
			}
			return ctx
		},
		OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*model.CallbackOutput]) context.Context {
			log.Printf("model end with stream output: %s", info.Name)
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("model error: %s", info.Name)
			return ctx
		},
	}
	toolHandler := &template_callbacks.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			toolCallID := compose.GetToolCallID(ctx)
			if debug {
				log.Printf("tool start: %s, tool call id: %s, input: %v", info.Name, toolCallID, input)
			} else {
				log.Printf("tool start: %s, tool call id: %s", info.Name, toolCallID)
			}
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			toolCallID := compose.GetToolCallID(ctx)
			if debug {
				log.Printf("tool end: %s, tool call id: %s, output: %v", info.Name, toolCallID, output)
			} else {
				log.Printf("tool end: %s, tool call id: %s", info.Name, toolCallID)
			}
			return ctx
		},
		OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*tool.CallbackOutput]) context.Context {
			toolCallID := compose.GetToolCallID(ctx)
			if debug {
				log.Printf("tool end: %s, tool call id: %s, output: %v", info.Name, toolCallID, output)
			} else {
				log.Printf("tool end: %s, tool call id: %s", info.Name, toolCallID)
			}
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			log.Printf("tool error: %s", info.Name)
			return ctx
		},
	}
	notifier := react.BuildAgentCallback(
		// model callback
		modelHandler,
		// tool callback
		toolHandler,
	)
	return notifier

	// handler := template_callbacks.NewHandlerHelper().ChatModel(modelHandler).Tool(toolHandler).Handler()

	// return handler
}

// buildThinkToolCallChecker 检查是否think需要调用工具
func buildThinkToolCallChecker() func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	return func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
		defer sr.Close()

		for {
			msg, err := sr.Recv()
			if err == io.EOF {
				return false, nil
			}
			if err != nil {
				return false, err
			}

			if len(msg.ToolCalls) > 0 {
				return true, nil
			}
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建MCPHub
	hub, err := einomcphost.NewMCPHub(ctx, `mcpservers.json`)
	if err != nil {
		log.Fatal(err)
	}
	defer hub.CloseServers()

	// 获取工具map
	toolsMap, err := hub.GetToolsMap(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(toolsMap)

	// 获取工具
	tools, err := hub.GetEinoTools(ctx, []string{"fofa_mcp_web_search", "fofa_mcp_fofa_web_crawler"})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(tools)

	openaiClient, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   "qwen3-0.6b",
		BaseURL: "http://127.0.0.1:1234/v1",
	})
	if err != nil {
		log.Fatal(err)
	}

	notifier := buildAgentCallback(true)
	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: openaiClient,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		StreamToolCallChecker: buildThinkToolCallChecker(),
	})
	if err != nil {
		log.Fatal(err)
	}
	s, err := reactAgent.Stream(ctx, []*schema.Message{
		{
			Role:    "user",
			Content: "搜索一下华顺信安公司成立于哪一年",
		},
	}, agent.WithComposeOptions(compose.WithCallbacks(notifier)))
	if err != nil {
		log.Fatal(err)
	}
	for {
		msg, err := s.Recv()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(msg.Content)
	}

}
