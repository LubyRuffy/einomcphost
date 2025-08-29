package llm

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// MustGetLmstudioModel 获取LMStudio模型，如果获取失败则panic
func MustGetLmstudioModel(ctx context.Context, modelName string) model.ToolCallingChatModel {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "http://127.0.0.1:1234/v1",
		Model:   modelName,
	})
	if err != nil {
		panic(err)
	}
	return chatModel
}
