package memory

import (
	"300Bot/conf"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	memoryTaskQueue      chan MemorySummary
	memoryWorkerInitOnce sync.Once
)

func init() {
	initMemoryWorkers()
}

func initMemoryWorkers() {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryBatchEnabled {
		return
	}
	memoryWorkerInitOnce.Do(func() {
		memoryTaskQueue = make(chan MemorySummary, conf.Memory.MemoryAsyncQueueSize)
		for i := 0; i < conf.Memory.MemoryWorkerCount; i++ {
			workerId := i + 1
			go memoryWorkerLoop(workerId)
		}
		log.Printf("memory worker started workers=%d queue=%d %s", conf.Memory.MemoryWorkerCount, conf.Memory.MemoryAsyncQueueSize, memoryMetricsLogKV())
	})
}

func EnqueueMemoryTask(summary MemorySummary) error {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryBatchEnabled {
		return nil
	}
	if strings.TrimSpace(summary.Scope) == "" {
		return fmt.Errorf("memory task scope 不能为空")
	}
	initMemoryWorkers()
	if memoryTaskQueue == nil {
		return fmt.Errorf("memory worker 队列未初始化")
	}
	select {
	case memoryTaskQueue <- summary:
		log.Printf("memory enqueue accepted scope=%s user=%s group=%s %s", summary.Scope, summary.UserId, summary.GroupId, memoryMetricsLogKV())
		return nil
	default:
		log.Printf("memory enqueue dropped scope=%s user=%s group=%s err=queue_full %s", summary.Scope, summary.UserId, summary.GroupId, memoryMetricsLogKV())
		return fmt.Errorf("memory worker 队列已满")
	}
}

func memoryWorkerLoop(workerId int) {
	for task := range memoryTaskQueue {
		processMemoryTask(workerId, task)
	}
}

func processMemoryTask(workerId int, summary MemorySummary) {
	retryTimes := conf.Memory.MemoryRetryTimes
	if retryTimes < 0 {
		retryTimes = 0
	}
	var err error
	for attempt := 0; attempt <= retryTimes; attempt++ {
		var dedupKey string
		dedupKey, err = UpsertMemorySummary(summary)
		if err == nil {
			memorySuccessCount.Add(1)
			log.Printf("memory upsert success worker=%d scope=%s user=%s group=%s dedup=%s %s", workerId, summary.Scope, summary.UserId, summary.GroupId, dedupKey, memoryMetricsLogKV())
			return
		}
		if attempt == retryTimes {
			break
		}
		backoff := memoryRetryBackoff(attempt)
		memoryRetryCount.Add(1)
		log.Printf("memory upsert retry worker=%d attempt=%d/%d backoff=%s err=%v %s", workerId, attempt+1, retryTimes, backoff, err, memoryMetricsLogKV())
		time.Sleep(backoff)
	}

	memoryFailureCount.Add(1)
	log.Printf("memory upsert degraded-to-l1 worker=%d scope=%s user=%s group=%s retries=%d err=%v fallback_to_mysql=%v %s", workerId, summary.Scope, summary.UserId, summary.GroupId, retryTimes, err, conf.Memory.MemoryFallbackToMysql, memoryMetricsLogKV())
	if conf.Memory.MemoryFallbackToMysql {
		SaveFallback(summary)
	}
}

func memoryRetryBackoff(attempt int) time.Duration {
	backoff := 500 * time.Millisecond
	for i := 0; i < attempt; i++ {
		backoff *= 2
		if backoff >= 8*time.Second {
			return 8 * time.Second
		}
	}
	return backoff
}
