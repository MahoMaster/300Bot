// Package inline 提供 LLM 交互 JSON 化（阶段四）的纯函数工具：
// 输出协议解析与兜底、记忆候选规范化。不依赖 conf 与 memory 父包，便于单元测试。
package inline

import (
	"encoding/json"
	"strings"
)

// 记忆候选约束：每条截断至 500 rune、每次回复最多 5 条
const (
	MaxCandidateRunes = 500
	MaxCandidates     = 5
)

// ChatReply 输出协议固定 schema：should_reply 为自主接话留扩展位（当前显式触发恒回复），
// memory 为顺带提取的记忆候选
type ChatReply struct {
	ShouldReply bool     `json:"should_reply"`
	Reply       string   `json:"reply"`
	Memory      []string `json:"memory"`
}

// ParseReply 解析 LLM 输出为 ChatReply。
// 兜底：任何解析失败（非 JSON / 无大括号 / unmarshal 出错）都把整段文本当作 reply 直接返回，
// 绝不让用户等空（路线图 7.3）
func ParseReply(raw string) ChatReply {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ChatReply{}
	}
	body := extractJSONBody(raw)
	if body == "" {
		return ChatReply{ShouldReply: true, Reply: raw}
	}
	var parsed ChatReply
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return ChatReply{ShouldReply: true, Reply: raw}
	}
	return parsed
}

// NormalizeReply 清洗解析结果：reply 与 memory 逐项 trim、丢弃空串、
// memory 每项截断至 MaxCandidateRunes、最多保留 MaxCandidates 条、去重
func NormalizeReply(r ChatReply) ChatReply {
	r.Reply = strings.TrimSpace(r.Reply)
	if len(r.Memory) == 0 {
		r.Memory = nil
		return r
	}
	seen := make(map[string]struct{}, len(r.Memory))
	cleaned := make([]string, 0, len(r.Memory))
	for _, item := range r.Memory {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if runes := []rune(item); len(runes) > MaxCandidateRunes {
			item = string(runes[:MaxCandidateRunes])
		}
		if _, dup := seen[item]; dup {
			continue
		}
		seen[item] = struct{}{}
		cleaned = append(cleaned, item)
		if len(cleaned) >= MaxCandidates {
			break
		}
	}
	if len(cleaned) == 0 {
		cleaned = nil
	}
	r.Memory = cleaned
	return r
}

// extractJSONBody 剥离 markdown 围栏并定位首个完整 {...} 段（与 batcher 同款简化逻辑，独立实现）
func extractJSONBody(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	left := strings.Index(raw, "{")
	right := strings.LastIndex(raw, "}")
	if left < 0 || right < 0 || right <= left {
		return ""
	}
	return raw[left : right+1]
}
