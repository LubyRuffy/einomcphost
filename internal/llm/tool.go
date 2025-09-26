package llm

import (
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// MustTool 创建一个工具，如果获取失败则panic
func MustTool[T, D any](name, description string, f utils.InvokeFunc[T, D]) tool.BaseTool {
	t, err := utils.InferTool(name, description, f)
	if err != nil {
		panic(err)
	}
	return t
}
