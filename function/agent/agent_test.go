package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// fakeCompleter 脚本化响应：按调用次序依次回放，并记录每次请求供事后断言
type fakeCompleter struct {
	scripts []func(req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	reqs    []openai.ChatCompletionRequest
}

func (f *fakeCompleter) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	f.reqs = append(f.reqs, req)
	idx := len(f.reqs) - 1
	if idx >= len(f.scripts) {
		return openai.ChatCompletionResponse{}, fmt.Errorf("unexpected extra call %d", idx)
	}
	return f.scripts[idx](req)
}

// contentResp 构造普通文本回复响应
func contentResp(content string, tokens int) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, Content: content},
		}},
		Usage: openai.Usage{TotalTokens: tokens},
	}
}

// toolCallResp 构造携带单个工具调用的响应
func toolCallResp(callID, name, args string, tokens int) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleAssistant,
				ToolCalls: []openai.ToolCall{{
					ID:       callID,
					Type:     openai.ToolTypeFunction,
					Function: openai.FunctionCall{Name: name, Arguments: args},
				}},
			},
		}},
		Usage: openai.Usage{TotalTokens: tokens},
	}
}

// multiToolCallResp 构造携带多个工具调用的响应
func multiToolCallResp(calls []openai.ToolCall, tokens int) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, ToolCalls: calls},
		}},
		Usage: openai.Usage{TotalTokens: tokens},
	}
}

func echoRegistry(t *testing.T) *Registry {
	t.Helper()
	reg := NewRegistry()
	if err := reg.Register(NewEchoTool()); err != nil {
		t.Fatalf("register echo tool: %v", err)
	}
	return reg
}

func baseReq() openai.ChatCompletionRequest {
	return openai.ChatCompletionRequest{
		Model:    "test-model",
		User:     "test-user",
		Messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "hi"}},
	}
}

// 空注册表：恰调用一次原样返回，与无工具前行为一致
func TestRunEmptyRegistrySingleCall(t *testing.T) {
	f := &fakeCompleter{scripts: []func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error){
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return contentResp("hello", 10), nil
		},
	}}
	r := NewRunner(f, NewRegistry(), 0, 0) // 非法参数回落默认值
	res, err := r.Run(context.Background(), baseReq())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(f.reqs) != 1 {
		t.Fatalf("expected 1 call, got %d", len(f.reqs))
	}
	if res.Rounds != 1 || res.TotalTokens != 10 || res.HadToolCall {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Response.Choices[0].Message.Content != "hello" {
		t.Fatalf("unexpected reply: %q", res.Response.Choices[0].Message.Content)
	}
}

// 两轮工具调用后正常收尾：消息组装正确（assistant ToolCalls + tool 消息 + ToolCallID 对齐），tokens 累加
func TestRunMultiRoundToolCalls(t *testing.T) {
	f := &fakeCompleter{scripts: []func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error){
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return toolCallResp("c1", EchoToolName, `{"content":"你好"}`, 5), nil
		},
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return toolCallResp("c2", EchoToolName, `{"content":"世界"}`, 6), nil
		},
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return contentResp("done", 7), nil
		},
	}}
	reg := echoRegistry(t)
	r := NewRunner(f, reg, 4, time.Second)
	req := baseReq()
	req.Tools = reg.Tools()
	res, err := r.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Rounds != 3 || res.TotalTokens != 18 || !res.HadToolCall {
		t.Fatalf("unexpected result: %+v", res)
	}
	// 第二轮请求的消息序列：user + assistant(tool_calls c1) + tool(c1 结果)
	msgs := f.reqs[1].Messages
	if len(msgs) != 3 {
		t.Fatalf("round2 messages = %d, want 3", len(msgs))
	}
	if len(msgs[1].ToolCalls) != 1 || msgs[1].ToolCalls[0].ID != "c1" {
		t.Fatalf("round2 assistant tool_calls broken: %+v", msgs[1])
	}
	if msgs[2].Role != openai.ChatMessageRoleTool || msgs[2].ToolCallID != "c1" || msgs[2].Content != "你好" {
		t.Fatalf("round2 tool message broken: %+v", msgs[2])
	}
	// 第三轮请求应携带两轮工具痕迹：user + 2*(assistant+tool)
	if got := len(f.reqs[2].Messages); got != 5 {
		t.Fatalf("round3 messages = %d, want 5", got)
	}
	// 调用方传入的切片不被循环改写
	if got := len(req.Messages); got != 1 {
		t.Fatalf("caller messages mutated: %d", got)
	}
}

// 轮数上限：模型持续调工具，最后一次请求带 tool_choice=none 与收尾 system 消息
func TestRunRoundLimitForcedNone(t *testing.T) {
	f := &fakeCompleter{scripts: []func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error){
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return toolCallResp("a1", EchoToolName, `{"content":"x"}`, 1), nil
		},
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return toolCallResp("a2", EchoToolName, `{"content":"y"}`, 1), nil
		},
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return contentResp("final", 2), nil
		},
	}}
	reg := echoRegistry(t)
	r := NewRunner(f, reg, 2, time.Second)
	req := baseReq()
	req.Tools = reg.Tools()
	res, err := r.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Rounds != 3 {
		t.Fatalf("rounds = %d, want 3 (2 rounds + 1 forced)", res.Rounds)
	}
	last := f.reqs[2]
	if last.ToolChoice != "none" {
		t.Fatalf("forced call tool_choice = %v, want none", last.ToolChoice)
	}
	tail := last.Messages[len(last.Messages)-1]
	if tail.Role != openai.ChatMessageRoleSystem || !strings.Contains(tail.Content, "工具调用轮次已达上限") {
		t.Fatalf("forced call missing limit hint, tail = %+v", tail)
	}
	if res.Response.Choices[0].Message.Content != "final" {
		t.Fatalf("unexpected forced reply: %q", res.Response.Choices[0].Message.Content)
	}
}

// 未知工具 / 执行器报错 / 执行器 panic：均以文本结果回传模型，循环不崩
func TestRunToolFailuresFeedbackToModel(t *testing.T) {
	reg := echoRegistry(t)
	if err := reg.Register(Tool{Name: "failer", Run: func(ctx context.Context, argsJSON string) (string, error) {
		return "", errors.New("网络错误")
	}}); err != nil {
		t.Fatalf("register failer: %v", err)
	}
	if err := reg.Register(Tool{Name: "boom", Run: func(ctx context.Context, argsJSON string) (string, error) {
		panic("炸了")
	}}); err != nil {
		t.Fatalf("register boom: %v", err)
	}

	calls := []openai.ToolCall{
		{ID: "g1", Type: openai.ToolTypeFunction, Function: openai.FunctionCall{Name: "ghost", Arguments: "{}"}},
		{ID: "f1", Type: openai.ToolTypeFunction, Function: openai.FunctionCall{Name: "failer", Arguments: "{}"}},
		{ID: "b1", Type: openai.ToolTypeFunction, Function: openai.FunctionCall{Name: "boom", Arguments: "{}"}},
	}
	f := &fakeCompleter{scripts: []func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error){
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return multiToolCallResp(calls, 3), nil
		},
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return contentResp("ok", 1), nil
		},
	}}
	r := NewRunner(f, reg, 4, time.Second)
	req := baseReq()
	req.Tools = reg.Tools()
	res, err := r.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Rounds != 2 {
		t.Fatalf("rounds = %d, want 2", res.Rounds)
	}
	msgs := f.reqs[1].Messages
	expect := map[string]string{
		"g1": "工具不存在: ghost",
		"f1": "网络错误",
		"b1": "panic",
	}
	toolMsgs := msgs[len(msgs)-3:]
	for i, m := range toolMsgs {
		if m.Role != openai.ChatMessageRoleTool {
			t.Fatalf("msg %d not tool role: %+v", i, m)
		}
		want, ok := expect[m.ToolCallID]
		if !ok {
			t.Fatalf("unexpected tool_call_id %q", m.ToolCallID)
		}
		if !strings.Contains(m.Content, want) {
			t.Fatalf("tool result %q missing %q", m.Content, want)
		}
	}
}

// 重复调用去重：同名同参第二次不再执行执行器，回填提示文本
func TestRunDedupSameNameArgs(t *testing.T) {
	reg := NewRegistry()
	execCount := 0
	if err := reg.Register(Tool{Name: "cnt", Run: func(ctx context.Context, argsJSON string) (string, error) {
		execCount++
		return fmt.Sprintf("exec-%d", execCount), nil
	}}); err != nil {
		t.Fatalf("register cnt: %v", err)
	}
	calls := []openai.ToolCall{
		{ID: "d1", Type: openai.ToolTypeFunction, Function: openai.FunctionCall{Name: "cnt", Arguments: `{"k":"v"}`}},
		{ID: "d2", Type: openai.ToolTypeFunction, Function: openai.FunctionCall{Name: "cnt", Arguments: `{"k":"v"}`}},
	}
	f := &fakeCompleter{scripts: []func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error){
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return multiToolCallResp(calls, 2), nil
		},
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return contentResp("done", 1), nil
		},
	}}
	r := NewRunner(f, reg, 4, time.Second)
	req := baseReq()
	req.Tools = reg.Tools()
	if _, err := r.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}
	if execCount != 1 {
		t.Fatalf("executor ran %d times, want 1", execCount)
	}
	msgs := f.reqs[1].Messages
	first, second := msgs[len(msgs)-2], msgs[len(msgs)-1]
	if first.Content != "exec-1" {
		t.Fatalf("first tool result = %q, want exec-1", first.Content)
	}
	if second.Content != dupCallHint {
		t.Fatalf("dup tool result = %q, want dedup hint", second.Content)
	}
}

// 总预算耗尽：第二轮调用时 ctx 已取消，Run 返回错误
func TestRunBudgetExhausted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := &fakeCompleter{scripts: []func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error){
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			cancel() // 第一轮返回后总预算耗尽
			return toolCallResp("t1", EchoToolName, `{"content":"x"}`, 1), nil
		},
		func(openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return openai.ChatCompletionResponse{}, ctx.Err()
		},
	}}
	reg := echoRegistry(t)
	r := NewRunner(f, reg, 4, time.Second)
	req := baseReq()
	req.Tools = reg.Tools()
	res, err := r.Run(ctx, req)
	if err == nil {
		t.Fatal("expected budget error, got nil")
	}
	if res.Rounds != 2 {
		t.Fatalf("rounds = %d, want 2", res.Rounds)
	}
}
