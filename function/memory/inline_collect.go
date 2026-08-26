package memory

import (
	"300Bot/conf"
	"log"
	"strings"
)

// EnqueueInlineCandidates 将回复时顺带提取的记忆候选（阶段四输出协议 memory 字段）
// 直接入记忆任务队列，与批量总结互补（高频轻量 + 低频深度，路线图 7.2）。
// user scope 锚定 QQ 号，group scope 不绑定单一 user_id（P11，与 buildMemorySummary 同款隔离）。
func EnqueueInlineCandidates(scope, userId, groupId, sessionId, messageId, source string, candidates []string) {
	if !conf.Memory.MemoryEnabled || !conf.Config.ChatMemoryInlineEnabled {
		return
	}
	scope = strings.ToLower(strings.TrimSpace(scope))
	source = strings.TrimSpace(source)
	if source == "" {
		source = "inline"
	}
	for _, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			continue
		}
		summary := MemorySummary{
			Scope:      scope,
			SessionId:  strings.TrimSpace(sessionId),
			MessageId:  strings.TrimSpace(messageId),
			Source:     source,
			Text:       "内联提取：" + cand,
			Summary:    cand,
			Tags:       []string{"inline"},
			Importance: 3,
			Confidence: 0.6,
			CreatedAt:  nowUnix(),
		}
		if scope == "user" {
			summary.UserId = strings.TrimSpace(userId)
		} else {
			summary.GroupId = strings.TrimSpace(groupId)
		}
		if err := EnqueueMemoryTask(summary); err != nil {
			log.Printf("memory inline enqueue failed scope=%s err=%v", scope, err)
		}
	}
}
