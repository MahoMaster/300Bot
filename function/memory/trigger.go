package memory

import (
	"strings"
	"sync"
	"time"
)

// trigger.go 事件触发：明确的身份声明/纠正/记忆指令类消息（"以后叫我XX"、"我搬到…"、"记住…"）
// 不应等 24 句批量阈值，命中关键词立即强制提取。关键词预筛为纯字符串匹配（微秒级、零 IO、不碰 LLM），
// 热路径开销可忽略；命中只负责"提前"，定性仍走提取+裁决两道 LLM 关卡。

// 强制提取前的落库等待：raw writer 按 1 秒定时 flush，等 3 秒确保触发消息本身已进库可被转写覆盖；
// 等待在独立 goroutine 内，热路径零阻塞
const memoryEventTriggerFlushDelay = 3 * time.Second

// eventTriggerLastFire 记录每个 owner 上次事件触发时间（unix 秒），用于冷却防刷屏
var eventTriggerLastFire sync.Map

// MatchEvent 关键词预筛（纯函数）：任一关键词出现即命中；空词表/空文本不命中
func MatchEvent(text string, keywords []string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" && strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// eventTriggerAllowed 同 owner 冷却判定：冷却期内重复命中（如刷屏"记住我说的话"）只触发一次。
// 尽力而为语义：极端并发下可能短暂双触发，由后续提取门槛与裁决消化，不追求互斥精度
func eventTriggerAllowed(ownerKey string, cooldownSec int) bool {
	if cooldownSec <= 0 {
		return true
	}
	now := nowUnix()
	if last, ok := eventTriggerLastFire.Load(ownerKey); ok {
		if lastUnix, isInt := last.(int64); isInt && now-lastUnix < int64(cooldownSec) {
			return false
		}
	}
	eventTriggerLastFire.Store(ownerKey, now)
	return true
}
