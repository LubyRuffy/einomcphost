package llm

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestMustGetGeminiModel(t *testing.T) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY is not set")
		return
	}

	model := MustGetGeminiModel(context.Background(), "models/gemini-2.5-flash", os.Getenv("GEMINI_API_KEY"))
	resp, err := model.Generate(context.Background(), []*schema.Message{
		{
			Role:    "user",
			Content: "Hello, how are you?",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp.Content)
	t.Log(resp.Content)
	fmt.Println(resp.Content)
}
