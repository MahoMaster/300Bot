package agent

import (
	"context"
	"encoding/json"
	"strings"
)

// EchoToolName echo 测试工具注册名
const EchoToolName = "echo"

// NewEchoTool 联调测试专用工具：原样返回输入内容。默认不注册，
// 由 chatGPT 包按配置（agentEchoToolEnabled）决定是否加入注册表
func NewEchoTool() Tool {
	return Tool{
		Name:        EchoToolName,
		Description: "联调测试专用工具：原样返回输入内容，正式对话中请勿调用。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "需要原样返回的内容",
				},
			},
			"required": []string{"content"},
		},
		Run: func(ctx context.Context, argsJSON string) (string, error) {
			if strings.TrimSpace(argsJSON) == "" {
				argsJSON = "{}"
			}
			var args struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", err
			}
			return strings.TrimSpace(args.Content), nil
		},
	}
}
