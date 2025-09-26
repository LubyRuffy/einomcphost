package reactrunner

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	template_callbacks "github.com/cloudwego/eino/utils/callbacks"
)

// LLMCallbacks 是react agentrunner llm的回调
type LLMCallbacks struct {
	// 工具调用
	OnToolCallStart func(toolCallID, toolName string, argumentsInJSON string) // 工具调用开始，流式的也只会调用一次
	OnToolCallEnd   func(toolCallID, toolName string)                         // 工具调用结束，流式的也只会调用一次

	// 推理内容返回
	OnReasoning func(reasoning string) // 推理内容返回，流式的就会调用多次
	OnResponse  func(response string)  // 响应内容返回，流式的就会调用多次

	// 调用错误
	OnError func(err error) // 调用错误
}

// ReactRunner 是react agentrunner
// 解决：实时跟踪工具调用，实时打印reasoning，实时打印response content
type ReactRunner struct {
	SystemPrompt string                     // 系统提示词
	Model        model.ToolCallingChatModel // 模型
	Tools        []tool.BaseTool            // 工具
	// MessageModifierFn func(ctx context.Context, input []*schema.Message) []*schema.Message // 消息修改器，用户进行消息替换处理
	LLMCallbacks       *LLMCallbacks                                                                                     // llm回调
	CompressToolMap    map[string]func(context.Context, []*schema.Message, int, *schema.Message) (*schema.Message, bool) // 压缩map, tool->compress func
	compressContentMap map[string]*schema.Message                                                                        // 压缩内容map, tool->compress func
}

// StreamToolCallCheckerfunc 流式输出工具调用检查器
// 兼容stream方式调用工具的场景
func StreamToolCallCheckerfunc(ctx context.Context, modelOutput *schema.StreamReader[*schema.Message]) (ret bool, err error) {
	defer modelOutput.Close()

	for {
		msg, err := modelOutput.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		if len(msg.ReasoningContent) > 0 {
			fmt.Print(msg.ReasoningContent)
		}

		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
}

// messageModifier 消息修改器
// 最初主要用于在消息中添加系统提示词；
// 也可以通过MessageModifierFn可以进行消息替换的场景：比如工具调用后，对工具返回的结果进行压缩
func (r *ReactRunner) messageModifier(ctx context.Context, input []*schema.Message) []*schema.Message {
	messages := make([]*schema.Message, 0, len(input)+1)
	messages = append(messages, schema.SystemMessage(r.SystemPrompt))

	for i, message := range input {
		// 如果压缩内容map中存在，则直接使用
		if compressContent, ok := r.compressContentMap[message.Content]; ok {
			message = compressContent
			messages = append(messages, message)
			continue
		}

		// 如果压缩toolmap中存在，则使用压缩函数进行压缩
		if compressFunc, ok := r.CompressToolMap[message.ToolName]; ok {
			// 压缩
			newMessage, changed := compressFunc(ctx, input, i, message)
			if changed {
				r.compressContentMap[message.Content] = newMessage
			}
			messages = append(messages, newMessage)
			continue
		}

		messages = append(messages, message)
	}
	return messages
}

// buildNotifier 构建notifier
func (r *ReactRunner) buildNotifier() callbacks.Handler {
	if r.LLMCallbacks == nil {
		r.LLMCallbacks = &LLMCallbacks{
			OnToolCallStart: func(toolCallID, toolName string, argumentsInJSON string) {
				log.Printf("toolCallStart: %s, %s, %s", toolCallID, toolName, argumentsInJSON)
			},
			OnToolCallEnd: func(toolCallID, toolName string) {
				log.Printf("toolCallEnd: %s, %s", toolCallID, toolName)
			},
			OnReasoning: func(reasoning string) {
				fmt.Printf("%s", reasoning)
			},
			OnResponse: func(response string) {
				fmt.Printf("%s", response)
			},
			OnError: func(err error) {
				log.Printf("error: %v", err)
			},
		}
	}
	return react.BuildAgentCallback(
		&template_callbacks.ModelCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
				return ctx
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
				if len(output.Message.ReasoningContent) > 0 {
					r.LLMCallbacks.OnReasoning(output.Message.ReasoningContent)
				}
				if len(output.Message.Content) > 0 {
					r.LLMCallbacks.OnResponse(output.Message.Content)
				}
				return ctx
			},
			OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*model.CallbackOutput]) context.Context {
				go func() {
					defer func() {
						if err := recover(); err != nil {
							log.Printf("[OnEndStream] panic err: %v", err)
						}
					}()

					defer output.Close()

					for {
						msg, err := output.Recv()
						if err != nil {
							return
						}
						if len(msg.Message.ReasoningContent) > 0 {
							r.LLMCallbacks.OnReasoning(msg.Message.ReasoningContent)
						}
						if len(msg.Message.Content) > 0 {
							r.LLMCallbacks.OnResponse(msg.Message.Content)
						}
					}
				}()
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				r.LLMCallbacks.OnError(err)
				return ctx
			},
		},
		&template_callbacks.ToolCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
				toolCallID := compose.GetToolCallID(ctx)
				r.LLMCallbacks.OnToolCallStart(toolCallID, info.Name, input.ArgumentsInJSON)
				return ctx
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
				toolCallID := compose.GetToolCallID(ctx)
				r.LLMCallbacks.OnToolCallEnd(toolCallID, info.Name)
				return ctx
			},
			OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*tool.CallbackOutput]) context.Context {
				toolCallID := compose.GetToolCallID(ctx)
				r.LLMCallbacks.OnToolCallEnd(toolCallID, info.Name)
				go func() {
					defer func() {
						if err := recover(); err != nil {
							log.Printf("[OnEndStream] panic err: %v", err)
						}
					}()

					defer output.Close()

					for {
						msg, err := output.Recv()
						if err != nil {
							return
						}
						if len(msg.Response) > 0 {
							r.LLMCallbacks.OnResponse(msg.Response)
						}
						if len(msg.Extra["reasoning"].(string)) > 0 {
							r.LLMCallbacks.OnReasoning(msg.Extra["reasoning"].(string))
						}
					}
				}()
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				r.LLMCallbacks.OnError(err)
				return ctx
			},
		},
	)
}

func (r *ReactRunner) newReactAgent(ctx context.Context) (*react.Agent, agent.AgentOption, error) {
	// 初始化压缩内容map
	if r.compressContentMap == nil {
		r.compressContentMap = make(map[string]*schema.Message, 0)
	}

	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: r.Model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: r.Tools,
		},
		StreamToolCallChecker: StreamToolCallCheckerfunc,
		MessageModifier:       r.messageModifier,
	})
	if err != nil {
		return nil, agent.AgentOption{}, err
	}

	notifier := r.buildNotifier()
	options := agent.WithComposeOptions(
		compose.WithCallbacks(notifier),
		compose.WithChatModelOption(
			openai.WithReasoningEffort(openai.ReasoningEffortLevelLow), // 低推理，提高响应速度
		),
	)

	return reactAgent, options, nil
}

// Run 运行react agent
func (r *ReactRunner) Run(ctx context.Context, messages []*schema.Message, stream bool) (string, string, error) {

	reactAgent, options, err := r.newReactAgent(ctx)
	if err != nil {
		return "", "", err
	}

	thinkReasoning := ""
	response := ""

	if stream {
		out, err := reactAgent.Stream(ctx,
			messages,
			options,
		)
		if err != nil {
			return "", "", err
		}
		defer out.Close()
		for {
			msg, err := out.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", "", err
			}

			thinkReasoning += msg.ReasoningContent
			response += msg.Content
		}
	} else {
		out, err := reactAgent.Generate(ctx,
			messages,
			options,
		)
		if err != nil {
			return "", "", err
		}
		thinkReasoning = out.ReasoningContent
		response = out.Content
	}
	return thinkReasoning, response, nil
}

func (r *ReactRunner) Stream(ctx context.Context, messages []*schema.Message, opts ...agent.AgentOption) (output *schema.StreamReader[*schema.Message], err error) {
	reactAgent, options, err := r.newReactAgent(ctx)
	if err != nil {
		return nil, err
	}
	return reactAgent.Stream(ctx, messages, options)
}

func (r *ReactRunner) Generate(ctx context.Context, messages []*schema.Message, opts ...agent.AgentOption) (output *schema.Message, err error) {
	reactAgent, options, err := r.newReactAgent(ctx)
	if err != nil {
		return nil, err
	}
	return reactAgent.Generate(ctx, messages, options)
}
