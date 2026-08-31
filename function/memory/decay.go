package memory

import (
	"300Bot/conf"
	"context"
	"log"
	"time"
)

// decay.go 记忆生命周期：不是所有记忆都应该永久存在。阶段性目标会失效、
// 长期未被再次验证的记忆应降低置信度直至过期。纯后台批处理（每日低峰），
// 绝不顺带在写路径执行（写路径保持最短）；仅处理条目型点（schema=entry_v1），
// 存量 legacy 点无该字段天然不受影响。

// 衰减规则常量：长期未验证先降置信度（-0.1），降到下限仍无新证据则置过期
const (
	memoryDecayWeakenStep      = 0.1
	memoryDecayConfidenceFloor = 0.3
)

// DecayMemories cron 入口：分页扫描两个集合的条目型记忆，
// goal 类超过 valid_until（若提取侧给出）或长期未更新（超过 memoryDecayStaleDays）的点：
// 低重要度（importance≤2）直接过期；其余先降置信度，降到下限后再过期。
// 全部走 SetPayloads 元数据更新，零重新 embed；过期为软删（召回 must_not 过滤），可回滚。
func DecayMemories() {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryLifecycleEnabled {
		return
	}
	repo, err := GetQdrantRepository()
	if err != nil {
		log.Printf("memory decay skipped: qdrant 不可用 err=%v", err)
		return
	}
	now := nowUnix()
	staleCutoff := now - int64(conf.Memory.MemoryDecayStaleDays)*86400
	expired, weakened := 0, 0
	for _, scope := range []string{"user", "group"} {
		scopeExpired, scopeWeakened := decayOneScope(repo, scope, now, staleCutoff)
		expired += scopeExpired
		weakened += scopeWeakened
	}
	if expired > 0 || weakened > 0 {
		log.Printf("memory decay done stale_days=%d expired=%d weakened=%d", conf.Memory.MemoryDecayStaleDays, expired, weakened)
	}
}

// decayOneScope 单集合分页扫描与处置；单页失败即中止本集合（下轮重来），避免带着错误状态继续翻页
func decayOneScope(repo *QdrantRepository, scope string, now int64, staleCutoff int64) (int, int) {
	expired, weakened := 0, 0
	offset := ""
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		records, next, err := repo.ScrollDecayBatch(ctx, scope, offset, 200)
		cancel()
		if err != nil {
			log.Printf("memory decay scroll failed scope=%s offset=%s err=%v", scope, offset, err)
			return expired, weakened
		}
		for _, rec := range records {
			action, kv := decideDecayAction(rec, now, staleCutoff)
			if action == "" {
				continue
			}
			if err := applyDecay(repo, scope, rec, kv); err != nil {
				log.Printf("memory decay apply failed scope=%s point=%s action=%s err=%v", scope, rec.Id, action, err)
				continue
			}
			if action == memoryStatusExpired {
				expired++
			} else {
				weakened++
			}
		}
		if next == "" || len(records) == 0 {
			return expired, weakened
		}
		offset = next
	}
}

// decideDecayAction 判定单点处置：返回动作（""=不动 / expired / weaken）与对应 payload 变更
func decideDecayAction(rec MemoryPointRecord, now int64, staleCutoff int64) (string, map[string]interface{}) {
	if rec.PayloadString("status") != memoryStatusActive {
		return "", nil
	}
	// goal 类显式有效期到期（提取侧后续可输出 valid_until；当前缺省 0 不走此分支）
	if validUntil := rec.PayloadInt64("valid_until"); validUntil > 0 && now > validUntil {
		return memoryStatusExpired, map[string]interface{}{"status": memoryStatusExpired, "updated_at": now}
	}
	updatedAt := rec.PayloadInt64("updated_at")
	if updatedAt <= 0 {
		updatedAt = rec.PayloadInt64("created_at")
	}
	if updatedAt <= 0 || updatedAt >= staleCutoff {
		return "", nil
	}
	// 长期未验证：低重要度直接过期；高价值记忆先降置信度，保留重新激活的机会
	if rec.PayloadInt64("importance") <= 2 {
		return memoryStatusExpired, map[string]interface{}{"status": memoryStatusExpired, "updated_at": now}
	}
	confidence := rec.PayloadFloat("confidence")
	if confidence <= memoryDecayConfidenceFloor+1e-9 {
		return memoryStatusExpired, map[string]interface{}{"status": memoryStatusExpired, "updated_at": now}
	}
	next := confidence - memoryDecayWeakenStep
	if next < memoryDecayConfidenceFloor {
		next = memoryDecayConfidenceFloor
	}
	return "weaken", map[string]interface{}{"confidence": next}
}

// applyDecay 执行单点元数据变更（SetPayloads 零 embed）；
// owner 从点自身 payload 读取（user_id/group_id），与 SetPayloads 的 collectionByScope 校验对齐
func applyDecay(repo *QdrantRepository, scope string, rec MemoryPointRecord, kv map[string]interface{}) error {
	userId := rec.PayloadString("user_id")
	groupId := rec.PayloadString("group_id")
	return repo.SetPayloads(scope, userId, groupId, []string{rec.Id}, kv)
}
