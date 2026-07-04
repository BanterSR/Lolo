package mcp

import (
	"github.com/bytedance/sonic"
	"github.com/openai/openai-go/v3"
)

// OpenAITools 以 openai 兼容格式返回全部工具定义
//
// command 中注册的 AI 无需改动 command 包即可接入,示例:
//
//	if s := mcp.Default(); s != nil {
//	    params.Tools = s.OpenAITools()
//	}
//	// ... 拿到 completion 后:
//	for _, tc := range completion.Choices[0].Message.ToolCalls {
//	    out := mcp.Default().CallTool(tc.Function.Name, tc.Function.Arguments)
//	    messages = append(messages, openai.ToolMessage(out, tc.ID))
//	}
//	// 再带着 messages 请求一次模型即可.
func (s *Server) OpenAITools() []openai.ChatCompletionToolUnionParam {
	list := make([]openai.ChatCompletionToolUnionParam, 0, len(s.toolNames))
	for _, name := range s.toolNames {
		t := s.toolMap[name]
		list = append(list, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  openai.FunctionParameters(t.InputSchema),
		}))
	}
	return list
}

// CallTool 按工具名与 json 入参执行工具,返回文本结果(供作为 tool 消息回填模型)
func (s *Server) CallTool(name, argumentsJSON string) string {
	t, ok := s.toolMap[name]
	if !ok {
		return "unknown tool: " + name
	}
	args := make(Args)
	if argumentsJSON != "" {
		if err := sonic.UnmarshalString(argumentsJSON, &args); err != nil {
			return "invalid arguments: " + err.Error()
		}
	}
	res, err := t.Handler(args)
	if err != nil {
		return err.Error()
	}
	return toText(res)
}
