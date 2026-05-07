package memory

import (
	"300Bot/conf"
	"300Bot/model"
	"log"
	"strconv"
	"strings"
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
		GroupId:   toIDString(msg["group_id"]),
		SessionId: strings.TrimSpace(sessionId),
		MessageId: messageId,
		Source:    source,
		InputText: rawText,
	}
	if turn.SessionId == "" || turn.MessageId == "" {
		return
	}
	if _, err := model.InsertMemoryRawTurn(turn); err != nil {
		log.Printf("CollectInput insert failed: %v", err)
		return
	}
	go TryBatchSummarizeOwner(turn.Scope, turn.UserId, turn.GroupId)
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
