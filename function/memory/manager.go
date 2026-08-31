package memory

import (
	"300Bot/conf"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// manager.go Memory Manager 裁决阶段：系统此前只有 Extractor（只会记）没有 Manager（不会改），
// 事实变化（住成都→搬上海）会堆积成矛盾记忆。这里补上：查旧记忆 → 裁决 → 执行。
// 优化目标与提取器分离：提取器尽量不漏，裁决器尽量不错。

// 重复验证的置信度演化规则：每次复现 +0.05，封顶 0.95；旧点无置信度时给 0.7 起步
const (
	memoryConfidenceBumpStep = 0.05
	memoryConfidenceCap      = 0.95
	memoryConfidenceFloor    = 0.7
)

// Reconcile 处理一条条目型记忆候选（调用前提：开关开启且 entry.IsEntry()）。
// 流程：同 owner 串行 → 精确三元组查旧记忆（无命中放宽到同 type）→
// 无旧记忆直接 ADD / 同值复现程序直决（零 LLM 零 embed）/ 否则单次裁决调用 → 程序执行决定。
// 任何裁决异常一律降级为 ADD（只增不删，安全方向）；返回错误交由 worker 重试/兜底链处理。
func Reconcile(entry MemorySummary) (string, error) {
	repo, err := GetQdrantRepository()
	if err != nil {
		return "", err
	}
	ownerId := selectOwnerID(entry.Scope, entry.UserId, entry.GroupId)
	if ownerId == "" {
		return "", fmt.Errorf("reconcile 缺少 owner：scope=%s", entry.Scope)
	}

	// 同 owner 串行：复用批量提取的 owner 锁，提取与裁决互斥，避免同 owner 并发改同一点
	lock := getOwnerLock(entry.Scope + ":" + ownerId)
	lock.Lock()
	defer lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), memoryLLMCallTimeoutSec*time.Second)
	defer cancel()

	// 1. 精确三元组查旧记忆（程序精确匹配优先，比纯向量搜索确定）
	olds, err := repo.ScrollEntries(ctx, entry.Scope, ownerId, entry.SubjectId, entry.Type, entry.Key, 8)
	if err != nil {
		log.Printf("memory manager scroll failed scope=%s owner=%s subject=%s key=%s err=%v (degrade to add)",
			entry.Scope, ownerId, entry.SubjectId, entry.Key, err)
		return UpsertMemorySummary(entry)
	}
	// 精确 key 无命中时放宽到同 type：捕捉同人同属性的近似旧记忆（仍带点 ID 可供裁决引用）
	if len(olds) == 0 {
		olds, err = repo.ScrollEntries(ctx, entry.Scope, ownerId, entry.SubjectId, entry.Type, "", 8)
		if err != nil {
			log.Printf("memory manager scroll widen failed scope=%s owner=%s subject=%s err=%v (degrade to add)",
				entry.Scope, ownerId, entry.SubjectId, err)
			return UpsertMemorySummary(entry)
		}
	}

	// 2. 无旧记忆：直接 ADD，省掉一次裁决调用（裁决 prompt 也约定空列表只能 add/ignore，
	// 价值判断已在提取门槛完成，这里不再重复把关）
	if len(olds) == 0 {
		log.Printf("memory manager add scope=%s owner=%s subject=%s type=%s key=%s reason=no_old", entry.Scope, ownerId, entry.SubjectId, entry.Type, entry.Key)
		return UpsertMemorySummary(entry)
	}

	// 3. 同值复现：同 key 旧点文本与候选一致 → 程序直决强化证据，零 LLM 零 embed
	for _, rec := range olds {
		if rec.PayloadString("mem_key") != entry.Key {
			continue
		}
		if normalizeMemoryText(rec.PayloadString("text")) != normalizeMemoryText(entry.Text) {
			continue
		}
		newCount := rec.PayloadInt64("evidence_count") + 1
		newConfidence := bumpMemoryConfidence(rec.PayloadFloat("confidence"))
		kv := map[string]interface{}{
			"evidence_count": newCount,
			"confidence":     newConfidence,
			"updated_at":     nowUnix(),
		}
		if err := repo.SetPayloads(entry.Scope, entry.UserId, entry.GroupId, []string{rec.Id}, kv); err != nil {
			return "", err
		}
		log.Printf("memory manager reinforced scope=%s owner=%s subject=%s key=%s point=%s evidence_count=%d confidence=%.2f",
			entry.Scope, ownerId, entry.SubjectId, entry.Key, rec.Id, newCount, newConfidence)
		return BuildEntryDedupKey(entry.Scope, ownerId, entry.SubjectId, entry.Type, entry.Key), nil
	}

	// 4. 冲突/相关：单次裁决调用
	decision, err := callAdjudication(entry, olds)
	if err != nil {
		log.Printf("memory manager adjudicate failed scope=%s owner=%s subject=%s key=%s err=%v (degrade to add)",
			entry.Scope, ownerId, entry.SubjectId, entry.Key, err)
		return UpsertMemorySummary(entry)
	}
	return executeDecision(repo, entry, olds, decision)
}

// executeDecision 程序执行裁决结果：update/merge 复用三元组键天然覆盖同点；删除一律软删可回滚
func executeDecision(repo *QdrantRepository, entry MemorySummary, olds []MemoryPointRecord, decision reconcileDecision) (string, error) {
	ownerId := selectOwnerID(entry.Scope, entry.UserId, entry.GroupId)
	switch decision.Decision {
	case "ignore":
		log.Printf("memory manager ignore scope=%s owner=%s subject=%s key=%s", entry.Scope, ownerId, entry.SubjectId, entry.Key)
		return "ignored", nil

	case "delete":
		target := findRecordById(olds, decision.TargetPointId)
		if target == nil {
			log.Printf("memory manager delete rejected: target not found scope=%s key=%s target=%s (degrade to add)", entry.Scope, entry.Key, decision.TargetPointId)
			return UpsertMemorySummary(entry)
		}
		kv := map[string]interface{}{"status": memoryStatusDeleted, "updated_at": nowUnix()}
		if err := repo.SetPayloads(entry.Scope, entry.UserId, entry.GroupId, []string{target.Id}, kv); err != nil {
			return "", err
		}
		log.Printf("memory manager soft-delete scope=%s owner=%s subject=%s key=%s point=%s", entry.Scope, ownerId, entry.SubjectId, entry.Key, target.Id)
		return "deleted:" + target.Id, nil

	case "update", "merge":
		target := findRecordById(olds, decision.TargetPointId)
		mergedValue := decision.MergedValue
		if target == nil || strings.TrimSpace(mergedValue) == "" {
			// 目标缺失或未给合并文本：降级 ADD，宁可多一条不可错改
			log.Printf("memory manager %s degraded-to-add: invalid target/value scope=%s key=%s target=%s", decision.Decision, entry.Scope, entry.Key, decision.TargetPointId)
			return UpsertMemorySummary(entry)
		}
		updated := entry
		updated.Text = strings.TrimSpace(mergedValue)
		updated.Summary = updated.Text
		// 证据累积：继承目标点已累积的证据次数 +1，避免合并后丢失置信度演化历史
		updated.EvidenceCount = int(target.PayloadInt64("evidence_count")) + 1
		if updated.EvidenceCount <= 1 {
			updated.EvidenceCount = 2
		}
		dedupKey, err := UpsertMemorySummary(updated)
		if err != nil {
			return "", err
		}
		// merge 时把同 key 的被并入点软删（保留可回滚）；update 不动其他点
		if decision.Decision == "merge" {
			absorbed := make([]string, 0, len(olds))
			for _, rec := range olds {
				if rec.Id == target.Id || rec.PayloadString("mem_key") != entry.Key {
					continue
				}
				absorbed = append(absorbed, rec.Id)
			}
			if len(absorbed) > 0 {
				kv := map[string]interface{}{"status": memoryStatusMerged, "updated_at": nowUnix()}
				if err := repo.SetPayloads(entry.Scope, entry.UserId, entry.GroupId, absorbed, kv); err != nil {
					log.Printf("memory manager merge soft-delete absorbed failed scope=%s key=%s err=%v (main point already merged)", entry.Scope, entry.Key, err)
				}
			}
		}
		log.Printf("memory manager %s scope=%s owner=%s subject=%s key=%s target=%s value=%s", decision.Decision, entry.Scope, ownerId, entry.SubjectId, entry.Key, target.Id, updated.Text)
		return dedupKey, nil

	default:
		// add 及未知决定：按新增处理
		log.Printf("memory manager add scope=%s owner=%s subject=%s type=%s key=%s reason=%s", entry.Scope, ownerId, entry.SubjectId, entry.Type, entry.Key, decision.Decision)
		return UpsertMemorySummary(entry)
	}
}

// callAdjudication 单次裁决调用：新候选 + 旧记忆列表一起给模型，输出五选一决定
func callAdjudication(entry MemorySummary, olds []MemoryPointRecord) (reconcileDecision, error) {
	client, err := getMemorySummaryClient()
	if err != nil {
		return reconcileDecision{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), memoryLLMCallTimeoutSec*time.Second)
	defer cancel()
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: conf.Memory.MemoryManagerModel,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: memoryReconcilePrompt},
				{Role: openai.ChatMessageRoleUser, Content: buildAdjudicationUserPrompt(entry, olds)},
			},
			User: selectOwnerID(entry.Scope, entry.UserId, entry.GroupId),
		},
	)
	if err != nil {
		return reconcileDecision{}, err
	}
	if len(resp.Choices) == 0 {
		return reconcileDecision{}, fmt.Errorf("empty choices")
	}
	return parseReconcileDecision(resp.Choices[0].Message.Content)
}

// buildAdjudicationUserPrompt 拼装裁决输入：候选 + 旧记忆（带点 ID 供 target_point_id 引用）
func buildAdjudicationUserPrompt(entry MemorySummary, olds []MemoryPointRecord) string {
	var b strings.Builder
	b.WriteString("新记忆候选：\n")
	fmt.Fprintf(&b, "subject_id=%s type=%s key=%s value=%s importance=%d confidence=%.2f\n\n",
		entry.SubjectId, entry.Type, entry.Key, entry.Summary, entry.Importance, entry.Confidence)
	b.WriteString("相关已有记忆：\n")
	for _, rec := range olds {
		fmt.Fprintf(&b, "[ID: %s] type=%s key=%s value=%s confidence=%.2f evidence_count=%d\n",
			rec.Id, rec.PayloadString("mem_type"), rec.PayloadString("mem_key"),
			rec.PayloadString("text"), rec.PayloadFloat("confidence"), rec.PayloadInt64("evidence_count"))
	}
	return b.String()
}

// findRecordById 校验裁决引用的目标点确在本轮旧记忆列表内，防止模型编造 ID 误改无关点
func findRecordById(records []MemoryPointRecord, pointId string) *MemoryPointRecord {
	pointId = strings.TrimSpace(pointId)
	if pointId == "" {
		return nil
	}
	for i := range records {
		if records[i].Id == pointId {
			return &records[i]
		}
	}
	return nil
}

// bumpMemoryConfidence 重复验证的置信度演化：旧值缺失给起步值，否则步进 +0.05 封顶 0.95
func bumpMemoryConfidence(old float64) float64 {
	if old <= 0 {
		return memoryConfidenceFloor
	}
	next := old + memoryConfidenceBumpStep
	if next > memoryConfidenceCap {
		next = memoryConfidenceCap
	}
	return next
}
