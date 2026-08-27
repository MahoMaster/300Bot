package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultMaxRounds   = 4
	defaultToolTimeout = 15 * time.Second

	// toolRoundLimitHint 达到轮数上限后的强制收尾指令
	toolRoundLimitHint = "工具调用轮次已达上限，请不要再调用任何工具，直接输出最终回复。"
	// dupCallHint 同名同参重复调用的回填结果，防模型死循环
	dupCallHint = "该调用与之前完全相同，请勿重复调用，请直接给出回复。"

	logPreviewRunes = 80
)

// Completer 最小 LLM 调用接口；*openai.Client 天然满足，单测用 fake 实现
type Completer interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// Runner 多轮工具调用循环执行器
type Runner struct {
	llm                Completer
	reg                *Registry
	maxRounds          int
	defaultToolTimeout time.Duration
}

// NewRunner 构造循环执行器；非法参数回落默认值（maxRounds=4、toolTimeout=15s）
func NewRunner(llm Completer, reg *Registry, maxRounds int, toolTimeout time.Duration) *Runner {
	if maxRounds <= 0 {
		maxRounds = defaultMaxRounds
	}
	if toolTimeout <= 0 {
		toolTimeout = defaultToolTimeout
	}
	return &Runner{
		llm:                llm,
		reg:                reg,
		maxRounds:          maxRounds,
		defaultToolTimeout: toolTimeout,
	}
}

// Result 一次请求的循环执行结果
type Result struct {
	Response    openai.ChatCompletionResponse // 最后一轮的响应
	TotalTokens int                           // 各轮 Usage.TotalTokens 累加
	Rounds      int                           // 实际 LLM 调用次数
	HadToolCall bool                          // 本次请求是否发生过工具调用
}

// Run 执行多轮循环：模型返回 tool_calls → 执行工具 → tool 消息回传 → 模型继续决策。
// 空注册表（或请求未携带 tools）时恰调用一次直接返回，与无工具前行为一致。
// 总时间预算由传入 ctx 控制（调用方用 LLMTimeoutSec 建超时），本函数不另起超时。
func (r *Runner) Run(ctx context.Context, req openai.ChatCompletionRequest) (Result, error) {
	var res Result
	// 无工具快速路径：单次调用，行为与旧逻辑完全一致
	if r.reg == nil || r.reg.Count() == 0 || len(req.Tools) == 0 {
		resp, err := r.llm.CreateChatCompletion(ctx, req)
		res.Response = resp
		res.TotalTokens = resp.Usage.TotalTokens
		res.Rounds = 1
		return res, err
	}

	loopMessages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+4)
	loopMessages = append(loopMessages, req.Messages...)
	// name+arguments 组合键去重，防模型重复发起相同调用
	called := make(map[string]struct{})

	for round := 1; round <= r.maxRounds; round++ {
		req.Messages = loopMessages
		start := time.Now()
		resp, err := r.llm.CreateChatCompletion(ctx, req)
		res.Rounds = round
		res.Response = resp
		res.TotalTokens += resp.Usage.TotalTokens
		if err != nil {
			return res, err
		}
		if len(resp.Choices) == 0 {
			return res, fmt.Errorf("empty choices")
		}
		msg := resp.Choices[0].Message
		log.Printf("agent round user=%s model=%s round=%d tool_calls=%d cost_ms=%d", req.User, req.Model, round, len(msg.ToolCalls), time.Since(start).Milliseconds())
		if len(msg.ToolCalls) == 0 {
			return res, nil
		}
		res.HadToolCall = true
		// assistant 消息（含 ToolCalls）进循环上下文；会话长期历史由调用方决定，不在此处扩散
		loopMessages = append(loopMessages, msg)
		for _, tc := range msg.ToolCalls {
			resultText := r.executeToolCall(ctx, req.User, tc, called)
			loopMessages = append(loopMessages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: tc.ID,
				Content:    resultText,
			})
		}
	}

	// 达到轮数上限仍在调工具：追加收尾指令并以 tool_choice=none 强制一次直接回复
	loopMessages = append(loopMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: toolRoundLimitHint,
	})
	req.Messages = loopMessages
	req.ToolChoice = "none"
	start := time.Now()
	resp, err := r.llm.CreateChatCompletion(ctx, req)
	res.Rounds++
	res.Response = resp
	res.TotalTokens += resp.Usage.TotalTokens
	log.Printf("agent round user=%s model=%s round=%d forced=none cost_ms=%d", req.User, req.Model, res.Rounds, time.Since(start).Milliseconds())
	if err != nil {
		return res, err
	}
	if len(resp.Choices) == 0 {
		return res, fmt.Errorf("empty choices")
	}
	return res, nil
}

// executeToolCall 执行单个工具调用：去重 → 查找 → 执行（子超时 + panic recover），
// 任何失败都转为结果文本回传模型，不中断循环
func (r *Runner) executeToolCall(ctx context.Context, user string, tc openai.ToolCall, called map[string]struct{}) string {
	name := tc.Function.Name
	args := tc.Function.Arguments

	dupKey := name + "\x00" + args
	if _, dup := called[dupKey]; dup {
		log.Printf("agent tool user=%s name=%s dup=true", user, name)
		return dupCallHint
	}
	called[dupKey] = struct{}{}

	tool, ok := r.reg.Get(name)
	if !ok {
		log.Printf("agent tool user=%s name=%s unknown=true", user, name)
		return "工具不存在: " + name
	}

	timeout := tool.Timeout
	if timeout <= 0 {
		timeout = r.defaultToolTimeout
	}
	// 子超时不得超过父 ctx 剩余预算
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		if remain := time.Until(deadline); remain < timeout {
			timeout = remain
		}
	}
	if timeout <= 0 {
		log.Printf("agent tool user=%s name=%s ok=false err=剩余时间预算不足", user, name)
		return "工具执行失败: 剩余时间预算不足"
	}
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	result, err := safeExecute(toolCtx, tool.Run, args)
	costMs := time.Since(start).Milliseconds()
	if err != nil {
		log.Printf("agent tool user=%s name=%s ok=false cost_ms=%d err=%v", user, name, costMs, err)
		return "工具执行失败: " + err.Error()
	}
	log.Printf("agent tool user=%s name=%s ok=true cost_ms=%d preview=%s", user, name, costMs, previewText(result, logPreviewRunes))
	return result
}

// safeExecute 执行器级兜底：panic 转 error，绝不让循环带 panic 出包（对齐 scheduler.safeRun 惯例）
func safeExecute(ctx context.Context, fn Executor, argsJSON string) (result string, err error) {
	defer func() {
		if info := recover(); info != nil {
			err = fmt.Errorf("执行器 panic: %v", info)
		}
	}()
	return fn(ctx, argsJSON)
}

// previewText 截断至 n 个 rune 作日志预览
func previewText(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
