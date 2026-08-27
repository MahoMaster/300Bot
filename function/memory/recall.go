package memory

import (
	"300Bot/conf"
	"300Bot/function/memory/recall"
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

// recallResult 召回产出：渲染文本（注入 system 段）+ 合并后命中（阶段四 JSON 化输入）
type recallResult struct {
	text string
	hits []recall.MemoryHit
}

// Recall 召回句柄：触发判定时刻创建并异步执行，worker 轮到执行时通过 Result/Hits 取结果。
// 预算在异步 goroutine 内强制执行，Result 永不阻塞——预算耗尽/尚未完成即降级为无记忆继续。
type Recall struct {
	ch   chan recallResult
	once sync.Once
	res  recallResult
}

// get 单次消费 channel 并缓存，保证 Result 与 Hits 取到同一份结果
func (r *Recall) get() recallResult {
	if r == nil {
		return recallResult{}
	}
	r.once.Do(func() {
		select {
		case r.res = <-r.ch:
		default:
		}
	})
	return r.res
}

// Result 返回召回渲染文本；未完成或已放弃时立即返回空串
func (r *Recall) Result() string {
	return r.get().text
}

// Hits 返回合并去重后的召回命中（已过滤阈值），用于结构化 JSON 注入；未完成时返回 nil
func (r *Recall) Hits() []recall.MemoryHit {
	return r.get().hits
}

func emptyRecall() *Recall {
	r := &Recall{ch: make(chan recallResult, 1)}
	r.ch <- recallResult{}
	return r
}

// StartRecall 在触发判定时刻（入队时）异步发起长期记忆召回，与排队等待重叠。
// userId 用于 user 集合过滤（QQ 号），groupId 用于 group 集合过滤，为空的一路不查。
func StartRecall(scope string, userId string, groupId string, query string) *Recall {
	query = strings.TrimSpace(query)
	if !conf.Memory.MemoryRecallEnabled || query == "" {
		return emptyRecall()
	}
	userId = strings.TrimSpace(userId)
	groupId = strings.TrimSpace(groupId)
	if userId == "" && groupId == "" {
		return emptyRecall()
	}
	repo, err := GetQdrantRepository()
	if err != nil {
		log.Printf("memory recall skipped: qdrant 不可用 err=%v", err)
		return emptyRecall()
	}
	_ = repo // 提前可用性检查；实际检索由 RecallSync 内部取库完成
	handle := &Recall{ch: make(chan recallResult, 1)}
	budget := time.Duration(conf.Memory.MemoryRecallBudgetMs) * time.Millisecond
	go runRecall(handle, scope, userId, groupId, query, budget)
	return handle
}

// RecallSync 同步召回：embedding 一次，user/group 两路并发检索，返回原始两路命中（未合并）。
// err 仅在 qdrant 不可用或 embedding 失败时非 nil；单路搜索失败只记日志、另一路结果照常返回。
// 时间预算由调用方通过 ctx 控制；userId/groupId 为空的一路不查。
// 供触发时刻异步召回（runRecall）与 recall_memory 工具（agenttool）共用，避免重复搜索逻辑。
func RecallSync(ctx context.Context, userId, groupId, query string) (userHits, groupHits []recall.MemoryHit, err error) {
	repo, err := GetQdrantRepository()
	if err != nil {
		return nil, nil, err
	}
	// embedding 只做一次，两 collection 共用同一向量
	vector, err := repo.EmbedQuery(query)
	if err != nil {
		return nil, nil, err
	}
	userId = strings.TrimSpace(userId)
	groupId = strings.TrimSpace(groupId)
	topK := conf.Memory.MemoryRecallTopK

	var (
		wg       sync.WaitGroup
		userErr  error
		groupErr error
	)
	if userId != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userHits, userErr = repo.Search(ctx, "user", userId, vector, topK)
		}()
	}
	if groupId != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			groupHits, groupErr = repo.Search(ctx, "group", groupId, vector, topK)
		}()
	}
	wg.Wait()

	// 部分降级：一路失败/超时用另一路；两路全失败则返回空命中由调用方处置
	if userId != "" && userErr != nil {
		log.Printf("memory recall user search failed owner=%s err=%v", userId, userErr)
	}
	if groupId != "" && groupErr != nil {
		log.Printf("memory recall group search failed owner=%s err=%v", groupId, groupErr)
	}
	return userHits, groupHits, nil
}

func runRecall(handle *Recall, scope, userId, groupId, query string, budget time.Duration) {
	defer func() {
		if info := recover(); info != nil {
			log.Printf("memory recall panic scope=%s info=%v", scope, info)
			select {
			case handle.ch <- recallResult{}:
			default:
			}
		}
	}()
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	userHits, groupHits, err := RecallSync(ctx, userId, groupId, query)
	if err != nil {
		log.Printf("memory recall degraded scope=%s stage=embed err=%v", scope, err)
		handle.ch <- recallResult{}
		return
	}

	hits := recall.MergeHits(userHits, groupHits, conf.Memory.MemoryRecallTopK, conf.Memory.MemoryRecallMinScore)
	text := recall.RenderText(hits, conf.Memory.MemoryRecallMaxChars)
	handle.ch <- recallResult{text: text, hits: hits}

	topScore := 0.0
	if len(hits) > 0 {
		topScore = hits[0].Score
	}
	log.Printf("memory recall scope=%s query_len=%d user_hits=%d group_hits=%d merged=%d top_score=%.2f cost_ms=%d preview=%s",
		scope, len([]rune(query)), len(userHits), len(groupHits), len(hits), topScore,
		time.Since(start).Milliseconds(), recall.PreviewText(text, 80))
}
