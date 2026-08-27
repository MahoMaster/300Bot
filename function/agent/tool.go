// Package agent 提供工具调用（function calling）地基：工具定义与注册表、
// 多轮 Agent 执行循环。按项目约定本包不依赖 conf/send/model，参数由调用方注入，便于单元测试。
package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// Executor 工具执行函数；argsJSON 为模型产出的参数 JSON 原文，
// 返回的字符串作为工具结果回传给模型
type Executor func(ctx context.Context, argsJSON string) (string, error)

// Tool 单个工具定义
type Tool struct {
	Name        string
	Description string
	Parameters  any           // JSON Schema，直接填入 FunctionDefinition.Parameters
	Timeout     time.Duration // 单次执行超时；<=0 时由 Runner 用默认超时
	Run         Executor
}

// Definition 转换为 OpenAI 协议 tools 字段格式
func (t Tool) Definition() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		},
	}
}

// Registry 工具注册表，并发安全，保持注册顺序
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
	order []string
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册工具；空名、缺执行函数、重名均报错
func (r *Registry) Register(t Tool) error {
	name := strings.TrimSpace(t.Name)
	if name == "" {
		return errors.New("agent: 工具名不能为空")
	}
	if t.Run == nil {
		return errors.New("agent: 工具 " + name + " 缺少执行函数")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[name]; ok {
		return errors.New("agent: 工具已存在 " + name)
	}
	t.Name = name
	r.tools[name] = t
	r.order = append(r.order, name)
	return nil
}

// Tools 按注册顺序返回协议格式工具列表；空注册表返回 nil（请求不携带 tools 字段）
func (r *Registry) Tools() []openai.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.order) == 0 {
		return nil
	}
	defs := make([]openai.Tool, 0, len(r.order))
	for _, name := range r.order {
		defs = append(defs, r.tools[name].Definition())
	}
	return defs
}

// Get 按名取工具
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Count 已注册工具数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.order)
}
