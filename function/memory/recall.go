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
	handle := &Recall{ch: make(chan recallResult, 1)}
	budget := time.Duration(conf.Memory.MemoryRecallBudgetMs) * time.Millisecond
	go runRecall(handle, repo, scope, userId, groupId, query, budget)
	return handle
}

func runRecall(handle *Recall, repo *QdrantRepository, scope, userId, groupId, query string, budget time.Duration) {
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

	// embedding 只做一次，两 collection 共用同一向量
	vector, err := repo.EmbedQuery(query)
	if err != nil || ctx.Err() != nil {
		log.Printf("memory recall degraded scope=%s stage=embed err=%v", scope, err)
		handle.ch <- recallResult{}
		return
	}

	var (
		wg        sync.WaitGroup
		userHits  []recall.MemoryHit
		groupHits []recall.MemoryHit
		userErr   error
		groupErr  error
	)
	topK := conf.Memory.MemoryRecallTopK
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

	// 部分降级：一路失败/超时用另一路；两路全失败则不带记忆继续
	if userId != "" && userErr != nil {
		log.Printf("memory recall user search failed owner=%s err=%v", userId, userErr)
	}
	if groupId != "" && groupErr != nil {
		log.Printf("memory recall group search failed owner=%s err=%v", groupId, groupErr)
	}

	hits := recall.MergeHits(userHits, groupHits, topK, conf.Memory.MemoryRecallMinScore)
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
