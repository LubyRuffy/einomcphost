package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func MustJson(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// ExtractJsonString 提取json字符串
func ExtractJsonString(str string) string {
	if strings.Contains(str, "```json") && strings.Count(str, "```") == 2 {
		// 提取中间部分
		str = strings.SplitN(str, "```json", 2)[1]
		str = strings.Trim(str, "```")
		str = strings.Trim(str, "\n\r\t ")
		return str
	}

	if strings.Count(str, "```") == 2 {
		// 提取中间部分
		newStr := strings.SplitN(str, "```", 2)[1]
		newStr = strings.Trim(newStr, "```")
		newStr = strings.Trim(newStr, "\n\r\t ")
		return newStr
	}

	toolRequestBegin := "[TOOL.Request]"
	if strings.Contains(str, toolRequestBegin) {
		// 提取中间部分
		str = strings.SplitN(str, toolRequestBegin, 2)[1]
		return strings.Split(str, "[END_TOOL.Request]")[0]
	}

	toolRequestBegin = "[TOOL_REQUEST]"
	if strings.Contains(str, toolRequestBegin) {
		// 提取中间部分
		str = strings.SplitN(str, toolRequestBegin, 2)[1]
		str = strings.Split(str, "[END_TOOL_REQUEST]")[0]

		if strings.HasPrefix(str, "{\n{") {
			str = "{" + strings.TrimLeft(str, "{\n{")
		}
		return str
	}

	thinkJson := "</think>\n{"
	if strings.Contains(str, thinkJson) {
		// 提取中间部分
		str = strings.SplitN(str, thinkJson, 2)[1]
		return "{" + str
	}

	// 这里尝试兼容think模型，硬编码做兼容，以后再改进
	if v := "\n{\""; strings.Contains(str, v) {
		// 提取中间部分
		position := strings.Index(str, v)
		return str[position+1:]
	}

	if v := "\n{\n\t\""; strings.Contains(str, v) {
		// 提取中间部分
		position := strings.Index(str, v)
		return str[position+1:]
	}

	return str
}

// gemini/glm的工具调用
type toolResponse struct {
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
	Delta        struct {
		Role        string            `json:"role"`
		Content     interface{}       `json:"content"`
		Audio       interface{}       `json:"audio"`
		ToolCalls   []schema.ToolCall `json:"tool_calls"`
		ToolCallId  interface{}       `json:"tool_call_id"`
		Attachments interface{}       `json:"attachments"`
		Metadata    interface{}       `json:"metadata"`
	} `json:"delta"`
}

type functionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func ExtractToolJson(str string) []schema.ToolCall {

	if strings.Contains(str, "delta") && strings.Contains(str, "tool_calls") {
		var tr toolResponse
		err := json.Unmarshal([]byte(str), &tr)
		if err == nil {
			return tr.Delta.ToolCalls
		}

	}

	if strings.Contains(str, "name") && strings.Contains(str, "arguments") {
		var t functionCall
		if err := json.Unmarshal([]byte(str), &t); err == nil {
			if t.Name != "" {
				return []schema.ToolCall{
					{
						Type: "function",
						Function: schema.FunctionCall{
							Name:      t.Name,
							Arguments: MustJson(t.Arguments),
						},
						ID: fmt.Sprintf("%d", rand.Int()),
					},
				}
			}
		}

		var t1 schema.FunctionCall
		if err := json.Unmarshal([]byte(str), &t1); err == nil {
			return []schema.ToolCall{
				{
					Type: "function",
					Function: schema.FunctionCall{
						Name:      t1.Name,
						Arguments: t1.Arguments,
					},
					ID: fmt.Sprintf("%d", rand.Int()),
				},
			}
		}
	}
	return nil
}
