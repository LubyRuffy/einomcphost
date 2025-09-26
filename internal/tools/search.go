package tools

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/tool/googlesearch"
	"github.com/cloudwego/eino/components/tool"
)

// NewGoogleSearchTool 调用google官方api的工具
func NewGoogleSearchTool(ctx context.Context) tool.BaseTool {
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	googleSearchEngineID := os.Getenv("GOOGLE_SEARCH_ENGINE_ID")

	if googleAPIKey == "" || googleSearchEngineID == "" {
		log.Fatal("[GOOGLE_API_KEY] and [GOOGLE_SEARCH_ENGINE_ID] must set")
	}

	// create tool
	searchTool, err := googlesearch.NewTool(ctx, &googlesearch.Config{
		APIKey:         googleAPIKey,
		SearchEngineID: googleSearchEngineID,
		Num:            10,
	})
	if err != nil {
		log.Fatal(err)
	}

	return searchTool
}
