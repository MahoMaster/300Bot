// Package recall 提供记忆召回的纯函数工具（命中结构、合并过滤、渲染、响应解析），
// 不依赖 conf 与 memory 父包，便于单元测试；召回编排见 memory 包 recall.go。
package recall

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// MemoryHit 召回命中的单条记忆
type MemoryHit struct {
	Score   float64
	Text    string
	Summary string
	Scope   string
}

// MergeHits 对两路召回结果做阈值过滤、每路 topK 截断、按文本去重，输出按分数降序
func MergeHits(userHits, groupHits []MemoryHit, topK int, minScore float64) []MemoryHit {
	pick := func(hits []MemoryHit) []MemoryHit {
		selected := make([]MemoryHit, 0, len(hits))
		for _, h := range hits {
			if h.Score < minScore {
				continue
			}
			if strings.TrimSpace(h.Text) == "" && strings.TrimSpace(h.Summary) == "" {
				continue
			}
			selected = append(selected, h)
		}
		if topK > 0 && len(selected) > topK {
			selected = selected[:topK]
		}
		return selected
	}
	merged := append(pick(userHits), pick(groupHits)...)
	// Qdrant 返回本身按分数降序，这里稳定排序保持两路内部顺序
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].Score > merged[j].Score })
	seen := make(map[string]struct{}, len(merged))
	deduped := make([]MemoryHit, 0, len(merged))
	for _, h := range merged {
		key := strings.TrimSpace(h.Text)
		if key == "" {
			key = strings.TrimSpace(h.Summary)
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, h)
	}
	return deduped
}

// RenderText 将命中渲染为注入 system 段的文本，按 rune 预算从高分到低分截断；
// 无可用命中返回空串
func RenderText(hits []MemoryHit, maxChars int) string {
	if len(hits) == 0 {
		return ""
	}
	header := "【关于对方的既有记忆】"
	budget := maxChars - len([]rune(header)) - 1
	if maxChars <= 0 || budget <= 0 {
		return ""
	}
	lines := make([]string, 0, len(hits))
	for _, h := range hits {
		text := strings.TrimSpace(h.Text)
		if text == "" {
			text = strings.TrimSpace(h.Summary)
		}
		if text == "" {
			continue
		}
		line := fmt.Sprintf("- (相似度%.2f) %s", h.Score, text)
		cost := len([]rune(line)) + 1
		if cost > budget {
			break
		}
		budget -= cost
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return header + "\n" + strings.Join(lines, "\n")
}

// ParseSearchResponse 解析 Qdrant /points/search 响应，只取召回关心的 payload 字段
func ParseSearchResponse(body []byte, scope string) ([]MemoryHit, error) {
	var resp struct {
		Result []struct {
			Score   float64 `json:"score"`
			Payload struct {
				Text    string `json:"text"`
				Summary string `json:"summary"`
			} `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	hits := make([]MemoryHit, 0, len(resp.Result))
	for _, item := range resp.Result {
		hits = append(hits, MemoryHit{
			Score:   item.Score,
			Text:    strings.TrimSpace(item.Payload.Text),
			Summary: strings.TrimSpace(item.Payload.Summary),
			Scope:   scope,
		})
	}
	return hits, nil
}

// PreviewText 截取前 maxRunes 个字符用于日志摘要
func PreviewText(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}
