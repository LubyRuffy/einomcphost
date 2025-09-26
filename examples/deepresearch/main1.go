package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/LubyRuffy/einomcphost/internal/llm"
	"github.com/LubyRuffy/einomcphost/internal/reactrunner"
	gotools "github.com/LubyRuffy/einomcphost/internal/tools"
	"github.com/LubyRuffy/einomcphost/internal/utils"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"
)

var (
	// defaultModel = llm.MustGetZhipuaiModel(context.Background(), "glm-4.5-flash", os.Getenv("ZHIPU_API"))
	defaultModel = llm.MustGetLmstudioModel(context.Background(), "mlx-community/gpt-oss-20b")
	// defaultModel = llm.MustGetLmstudioModel(context.Background(), "qwen3-0.6b")
)

type LoggerCallback struct {
	callbacks.HandlerBuilder // 可以用 callbacks.HandlerBuilder 来辅助实现 callback
}

func (cb *LoggerCallback) OnHandOff(ctx context.Context, info *host.HandOffInfo) context.Context {
	fmt.Printf("OnHandOff: %s, %s\n", info.ToAgentName, info.Argument)
	return ctx
}

func compressGoogleSearchResponse(ctx context.Context, input []*schema.Message, i int, message *schema.Message) (*schema.Message, bool) {
	rawRole := message.Role
	message.Role = schema.User
	defer func() {
		message.Role = rawRole
	}()

	msg, err := defaultModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(fmt.Sprintf(`你是一个知识渊博的信息提取专家，请根据用户的问题，和搜索引擎返回结果，对其进行相关性的筛选。

筛选主要分为两个方面:
- 如果有多个返回的描述是相同的，需要去除重复内容，只保留一个；注意，尽可能保留权威性更高的来源的内容；
- 如果搜索返回的内容跟主题相关性不大，需要丢弃。

返回的结果要跟搜索引擎返回的内容保持一致，且能够通过json.Unmarshal进行反序列化（不能带有任何多余的字符）。
{
	"query":"xxx",
	"results":[
		{
			"title":"xxx",
			"url":"https://reefresilience.org/zh-TW/bleaching/mass-bleaching/",
			"description":"xxx"
		}
	]
}

用户最初的问题是：
<root_question>%s</root_question>

目前用户进行搜索的语句是：
<search_query>%s</search_query>`, input[0].Content, input[i-1].Content)),
		message,
	})
	if err != nil {
		return message, false
	}

	jsonContent := utils.ExtractJsonString(msg.Content)

	var searchResponse gotools.SearchResponse
	if err := json.Unmarshal([]byte(jsonContent), &searchResponse); err != nil {
		return message, false
	}

	// 精简url
	// for i, result := range searchResponse.Results {
	// 	result.URL = fmt.Sprintf("http://u.io/id=%d", i)
	// }

	message.Content = utils.MustJson(searchResponse)
	return message, true
}

func compressFetchUrlResponse(ctx context.Context, input []*schema.Message, i int, message *schema.Message) (*schema.Message, bool) {
	msg, err := defaultModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(`你是一个知识渊博的信息提取专家，请根据用户的问题，和搜索引擎返回结果，对其进行内容的摘要压缩，只保留相关的内容，去掉不相关的文本，减少文本的长度。

返回的结果要跟搜索引擎返回的内容保持一致，且能够通过json.Unmarshal进行反序列化（不能带有任何多余的字符）。
{
	"title":"xxx",
	"content":"xxx"
}
`),
		message,
	})
	if err != nil {
		return message, false
	}

	jsonContent := utils.ExtractJsonString(msg.Content)

	var fetchUrlResponse gotools.FetchUrlResponse
	if err := json.Unmarshal([]byte(jsonContent), &fetchUrlResponse); err != nil {
		return message, false
	}

	return message, true
}

// 生成查询关键语句的Agent
func newWebSearchAgent(ctx context.Context) *host.Specialist {

	systemPrompt := `你是一个知识渊博的信息检索专家，请根据用户的问题，返回结果。今天是{{current_time}}。

规则：
- 在搜索的时候注意优先搜索最近的内容。`

	systemPrompt = strings.ReplaceAll(systemPrompt, "{{current_time}}", time.Now().Format("2006-01-02"))

	reactRunner := &reactrunner.ReactRunner{
		SystemPrompt: systemPrompt,
		Model:        defaultModel,
		Tools: []tool.BaseTool{
			// gotools.NewGoogleSearchTool(ctx),
			gotools.NewGoogleSearchLocal(),
			gotools.NewFetchUrlLocal(),
		},
		CompressToolMap: map[string]func(context.Context, []*schema.Message, int, *schema.Message) (*schema.Message, bool){
			"google_search": compressGoogleSearchResponse,
			"fetch_url":     compressFetchUrlResponse,
		},
	}

	return &host.Specialist{
		AgentMeta: host.AgentMeta{
			Name:        "web_search_agent",
			IntendedUse: "根据用户最初的问题以及中间的思考过程，针对当前的问题，生成结果。",
		},
		Streamable: reactRunner.Stream,
		Invokable:  reactRunner.Generate,
	}
}

func main() {
	stream := flag.Bool("stream", false, "use stream mode")
	flag.Parse()

	message := "马尔代夫的珊瑚为什么白化了？"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hostMA, err := host.NewMultiAgent(ctx, &host.MultiAgentConfig{
		Host: host.Host{
			ToolCallingModel: defaultModel,
			SystemPrompt:     "你是一个知识渊博的信息检索专家，请根据用户的问题，调用不同的工具，并把答案返回给用户。",
		},
		Specialists: []*host.Specialist{
			newWebSearchAgent(ctx), // 查询结果
		},
		StreamToolCallChecker: reactrunner.StreamToolCallCheckerfunc,
	})
	if err != nil {
		panic(err)
	}

	msg := &schema.Message{
		Role:    schema.User,
		Content: message,
	}
	cb := &LoggerCallback{}

	if *stream {
		out, err := hostMA.Stream(ctx, []*schema.Message{msg},
			host.WithAgentCallbacks(cb),
		)
		if err != nil {
			panic(err)
		}

		for {
			msg, err := out.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
			}

			print(msg.Content)
		}
	} else {
		out, err := hostMA.Generate(ctx, []*schema.Message{msg},
			host.WithAgentCallbacks(cb),
		)
		if err != nil {
			panic(err)
		}
		print(out.Content)
	}
}
