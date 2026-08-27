// Package agenttool 提供 Agent 循环的业务工具实现（如 recall_memory）。
// 按项目约定本包不依赖 conf/send/model，搜索函数与阈值全部构造时注入，便于单元测试。
package agenttool

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"300Bot/function/agent"
	"300Bot/function/memory/recall"
)

// RecallToolName recall_memory 工具名
const RecallToolName = "recall_memory"

// SearchFn 与 memory.RecallSync 同签名：embedding 一次、user/group 两路并发检索，返回原始两路命中
type SearchFn func(ctx context.Context, userId, groupId, query string) ([]recall.MemoryHit, []recall.MemoryHit, error)

// RecallToolOptions 召回工具执行所需依赖与阈值，由接线方注入
type RecallToolOptions struct {
	Search   SearchFn
	TopK     int
	MinScore float64
	MaxChars int
	Budget   time.Duration // 单次执行时间预算；<=0 时默认 2s
}

// RecallIdentity 当前发言人身份；由代码注入 ctx，而非模型传参，避免身份冒用
type RecallIdentity struct {
	UserQQ  string
	GroupID string
}

type recallIdentityKey struct{}

// WithRecallIdentity 将当前发言人身份写入 ctx，在 Agent 循环启动前调用
func WithRecallIdentity(ctx context.Context, id RecallIdentity) context.Context {
	return context.WithValue(ctx, recallIdentityKey{}, id)
}

func recallIdentityFrom(ctx context.Context) (RecallIdentity, bool) {
	id, ok := ctx.Value(recallIdentityKey{}).(RecallIdentity)
	return id, ok
}

// recallArgs 模型产出的参数结构
type recallArgs struct {
	Query string `json:"query"`
	Scope string `json:"scope"`
}

// NewRecallMemoryTool 构建 recall_memory 工具：按需检索当前发言人或本群的长期记忆。
// 除参数 JSON 解析失败返回 error 外，其余异常一律软文本提示，让模型平滑降级、不中断循环。
func NewRecallMemoryTool(opts RecallToolOptions) agent.Tool {
	budget := opts.Budget
	if budget <= 0 {
		budget = 2 * time.Second
	}
	return agent.Tool{
		Name: RecallToolName,
		Description: "检索关于当前发言人（scope=user）或本群（scope=group）的长期记忆，如过往事实、偏好、发生过的事。" +
			"对话环境中可能已提供部分记忆，仅当需要补充其他话题的信息时才调用；query 用简短短语。",
		Timeout: budget + time.Second, // 外层超时略大于内层预算，保证降级提示有机会回传
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "要检索的记忆内容，简短短语",
				},
				"scope": map[string]any{
					"type":        "string",
					"enum":        []string{"user", "group"},
					"description": "user 查当前发言人的记忆，group 查本群的记忆；省略则两路都查",
				},
			},
			"required": []string{"query"},
		},
		Run: func(ctx context.Context, argsJSON string) (string, error) {
			var args recallArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", err
			}
			query := strings.TrimSpace(args.Query)
			if query == "" {
				return "查询内容不能为空，请提供要检索的关键词", nil
			}
			id, ok := recallIdentityFrom(ctx)
			if !ok || (id.UserQQ == "" && id.GroupID == "") {
				return "当前上下文缺少发言人身份信息", nil
			}

			// 按 scope 解析要查的路；不查的一路传空串，由 Search 内部跳过
			var userId, groupId string
			switch strings.ToLower(strings.TrimSpace(args.Scope)) {
			case "":
				userId, groupId = id.UserQQ, id.GroupID
			case "user":
				if id.UserQQ == "" {
					return "当前上下文缺少发言人身份信息", nil
				}
				userId = id.UserQQ
			case "group":
				if id.GroupID == "" {
					return "当前是私聊，没有群记忆可查", nil
				}
				groupId = id.GroupID
			default:
				return "scope 仅支持 user 或 group，也可省略表示两路都查", nil
			}

			searchCtx, cancel := context.WithTimeout(ctx, budget)
			defer cancel()
			userHits, groupHits, err := opts.Search(searchCtx, userId, groupId, query)
			if err != nil {
				return "记忆检索服务暂时不可用，请基于已有信息回复", nil
			}
			hits := recall.MergeHits(userHits, groupHits, opts.TopK, opts.MinScore)
			if len(hits) == 0 {
				return "未检索到相关记忆。", nil
			}
			if text := recall.RenderText(hits, opts.MaxChars); text != "" {
				return text, nil
			}
			return "未检索到相关记忆。", nil
		},
	}
}
