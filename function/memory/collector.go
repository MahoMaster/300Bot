package memory

import (
	"300Bot/conf"
	"300Bot/function/chatctx"
	"300Bot/model"
	"log"
	"strconv"
	"strings"
	"time"
)

func CollectInput(scope string, source string, sessionId string, msg map[string]interface{}) {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryRawStoreEnabled {
		return
	}
	rawText := strings.TrimSpace(getString(msg, "raw_message"))
	if rawText == "" {
		return
	}
	messageId := toIDString(msg["message_id"])
	turn := model.MemoryRawTurn{
		Scope:     scope,
		UserId:    toIDString(msg["user_id"]),
		Nickname:  chatctx.SenderNickname(msg),
		GroupId:   toIDString(msg["group_id"]),
		SessionId: strings.TrimSpace(sessionId),
		MessageId: messageId,
		Source:    source,
		InputText: rawText,
	}
	if turn.SessionId == "" || turn.MessageId == "" {
		return
	}
	// 热路径零 DB 阻塞（P13）：仅非阻塞入队，批量落库后由 raw writer 触发总结；
	// 队列满仅记日志丢弃（raw_turns 本就是 L1 兜底，窗口上下文不受影响）
	enqueueRawTurn(turn)

	// 事件触发（开关）：身份声明/纠正/记忆指令类消息不等批量阈值，命中关键词且冷却通过即强制提取；
	// 纯字符串匹配零 IO，热路径开销可忽略；延迟在独立 goroutine 内等待落库，不阻塞消息管道
	if conf.Memory.MemoryEventTriggerEnabled && MatchEvent(rawText, conf.Memory.MemoryEventTriggerKeywords) {
		ownerKey := scope + ":" + selectOwnerID(scope, turn.UserId, turn.GroupId)
		if eventTriggerAllowed(ownerKey, conf.Memory.MemoryEventTriggerCooldownSec) {
			log.Printf("memory event trigger hit scope=%s owner=%s message=%s", scope, ownerKey, turn.MessageId)
			go func(triggerScope, triggerUserId, triggerGroupId string) {
				time.Sleep(memoryEventTriggerFlushDelay)
				TryBatchSummarizeOwnerForced(triggerScope, triggerUserId, triggerGroupId)
			}(scope, turn.UserId, turn.GroupId)
		}
	}
}

func CollectOutput(scope string, source string, sessionId string, msg map[string]interface{}, replyText string) {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryRawStoreEnabled {
		return
	}
	replyText = strings.TrimSpace(replyText)
	if replyText == "" {
		return
	}
	turn := model.MemoryRawTurn{
		Scope:     scope,
		UserId:    toIDString(msg["user_id"]),
		GroupId:   toIDString(msg["group_id"]),
		SessionId: strings.TrimSpace(sessionId),
		MessageId: toIDString(msg["message_id"]),
		Source:    source,
		ReplyText: replyText,
	}
	if turn.SessionId == "" || turn.MessageId == "" {
		return
	}
	if err := model.UpsertMemoryRawOutput(turn); err != nil {
		log.Printf("CollectOutput upsert failed: %v", err)
		return
	}
	go TryBatchSummarizeOwner(turn.Scope, turn.UserId, turn.GroupId)
}

func getString(msg map[string]interface{}, key string) string {
	val, ok := msg[key]
	if !ok || val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func toIDString(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.Itoa(int(v))
	default:
		return ""
	}
}
