package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractJsonString(t *testing.T) {
	assert.Equal(t, `{}`, ExtractJsonString("```json{}\n```"))
	assert.Equal(t, `{}`, ExtractJsonString("```json\n{}\n```"))
	assert.Equal(t, `{"a":1}`, ExtractJsonString("```json\n{\"a\":1}\n```"))
	assert.Equal(t, `{"a":1}`, ExtractJsonString("aaabbbccc```json\n{\"a\":1}\n```"))
	assert.Equal(t, `{"think":1}`, ExtractJsonString("aaabbbccc\n\n\n{\"think\":1}"))
	assert.Equal(t, `{"think":1}`, ExtractJsonString("aaabbbccc\n{\"think\":1}"))
	assert.Equal(t, "{\n\t\"think\":1}", ExtractJsonString("aaabbbccc\n{\n\t\"think\":1}"))
	assert.Equal(t, "{\n    \"think\": 1\n}", ExtractJsonString(" \n```json\n{\n    \"think\": 1\n}\n```"))
	assert.Equal(t, "{\n    \"think\": 1\n}", ExtractJsonString(" \n```\n{\n    \"think\": 1\n}\n```"))
	assert.Equal(t, "{\n    \"think\": 1\n}\n", ExtractJsonString("\n<think>\n好的\n</think>\n{\n    \"think\": 1\n}\n"))
	assert.Equal(t, "{\n\"think\":1\n}\n", ExtractJsonString(`[TOOL_REQUEST]{
"think":1
}
[END_TOOL_REQUEST]`))
	assert.Equal(t, `{"name": "url_markdown", "arguments": {"think": "aaa", "url": "bbb"}}`, ExtractJsonString("[TOOL_REQUEST]{\n{\"name\": \"url_markdown\", \"arguments\": {\"think\": \"aaa\", \"url\": \"bbb\"}}[END_TOOL_REQUEST]"))
}

func TestExtractToolJson(t *testing.T) {
	toolJson := `{"index":0,"finish_reason":"tool_calls","delta":{"role":"assistant","content":null,"audio":null,"tool_calls":[{"id":"call_202504181105130952561a4da84886_0","index":0,"type":"function","function":{"name":"url_markdown","arguments":"{\"url\": \"nosec.org\"}","outputs":null},"code_interpreter":null,"retrieval":null,"drawing_tool":null,"web_browser":null,"search_intent":null,"search_result":null}],"tool_call_id":null,"attachments":null,"metadata":null}}`
	tools := ExtractToolJson(toolJson)
	assert.True(t, len(tools) > 0)

	toolJson = `{
  "name" : "url_markdown",
  "arguments" : {
    "think" : "搜索结果中第一个链接看起来最有希望，标题明确包含'年报2023'。现在尝试访问该页面并提取内容。",
    "url" : "http://www.pbc.gov.cn/chubanwu/114566/115296/5378398/5378450/index.html"
  }
}`
	tools = ExtractToolJson(toolJson)
	assert.True(t, len(tools) > 0)

	toolJson = `{
  "name" : "url_markdown",
  "arguments" : "{\"think\" : \"搜索结果中第一个链接看起来最有希望，标题明确包含'年报2023'。现在尝试访问该页面并提取内容。\",\"url\" : \"http://www.pbc.gov.cn/chubanwu/114566/115296/5378398/5378450/index.html\"}"
}`
	tools = ExtractToolJson(toolJson)
	assert.True(t, len(tools) > 0)
}
