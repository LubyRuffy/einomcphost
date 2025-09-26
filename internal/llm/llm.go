// Copyright 2025 einomcp
//
// Package llm 提供LLM模型的获取方法
// 默认都启用openai兼容的API接口
package llm

import (
	"context"
	"net/http"
	"net/url"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

var (
	// DefaultLmstudioUrl 默认的LMStudio URL
	DefaultLmstudioUrl = "http://127.0.0.1:1234/v1"
	// DefaultOllamaUrl 默认的Ollama URL
	DefaultOllamaUrl = "http://127.0.0.1:11434/v1"
	// DefaultOpenaiUrl 默认的OpenAI URL
	DefaultOpenaiUrl = "https://api.openai.com/v1"
	// DefaultGeminiUrl 默认的Gemini URL
	DefaultGeminiUrl = "https://generativelanguage.googleapis.com/v1beta/openai"
	// DefaultZhipuaiUrl 默认的智谱AI URL
	DefaultZhipuaiUrl = "https://open.bigmodel.cn/api/paas/v4"

	// ProxyUrl 代理URL
	ProxyUrl = ""
	// ProxyUrl = "http://127.0.0.1:18080"
)

// MustGetOpenaiCompatibleModel 获取OpenAI兼容的模型，如果获取失败则panic
func MustGetOpenaiCompatibleModel(ctx context.Context, modelName string, baseUrl string, apiKey string) model.ToolCallingChatModel {
	var httpClient *http.Client
	if ProxyUrl != "" {
		proxyUrl, err := url.Parse(ProxyUrl)
		if err != nil {
			panic(err)
		}
		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:    baseUrl,
		Model:      modelName,
		APIKey:     apiKey,
		HTTPClient: httpClient,
	})
	if err != nil {
		panic(err)
	}
	return chatModel
}

// MustGetLmstudioModel 获取LMStudio模型，如果获取失败则panic
func MustGetLmstudioModel(ctx context.Context, modelName string) model.ToolCallingChatModel {
	return MustGetOpenaiCompatibleModel(ctx, modelName, DefaultLmstudioUrl, "")
}

// MustGetOllamaModel 获取Ollama模型，如果获取失败则panic
func MustGetOllamaModel(ctx context.Context, modelName string) model.ToolCallingChatModel {
	return MustGetOpenaiCompatibleModel(ctx, modelName, DefaultOllamaUrl, "")
}

// MustGetOpenaiModel 获取OpenAI模型，如果获取失败则panic
func MustGetOpenaiModel(ctx context.Context, modelName string, apiKey string) model.ToolCallingChatModel {
	return MustGetOpenaiCompatibleModel(ctx, modelName, DefaultOpenaiUrl, apiKey)
}

// MustGetGeminiModel 获取Gemini模型，如果获取失败则panic
// modelName类似于 models/gemini-2.5-pro
func MustGetGeminiModel(ctx context.Context, modelName string, apiKey string) model.ToolCallingChatModel {
	return MustGetOpenaiCompatibleModel(ctx, modelName, DefaultGeminiUrl, apiKey)
}

func MustGetZhipuaiModel(ctx context.Context, modelName string, apiKey string) model.ToolCallingChatModel {
	return MustGetOpenaiCompatibleModel(ctx, modelName, DefaultZhipuaiUrl, apiKey)
}
