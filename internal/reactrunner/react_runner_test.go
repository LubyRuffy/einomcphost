package reactrunner

import (
	"context"
	"testing"

	"github.com/LubyRuffy/einomcphost/internal/llm"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestReactRunner_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type WeatherInput struct {
		City string `json:"city" jsonschema:"required,description=城市"`
	}
	type WeatherOutput struct {
		Weather string `json:"weather" jsonschema:"required,description=天气"`
	}
	reactRunner := &ReactRunner{
		SystemPrompt: "你是一个天气专家，你会回答用户的天气问题。",
		Model:        llm.MustGetLmstudioModel(ctx, "mlx-community/gpt-oss-20b"),
		Tools: []tool.BaseTool{
			llm.MustTool("weather", "天气查询", func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
				return WeatherOutput{
					Weather: "今天天气晴朗，气温25度。",
				}, nil
			}),
		},
	}
	thinkReasoning, res, err := reactRunner.Run(ctx, []*schema.Message{
		schema.UserMessage("今天北京天气如何"),
	}, true)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotNil(t, thinkReasoning)
}
