package memory

import (
	"fmt"
	"sync/atomic"
)

var (
	memorySuccessCount atomic.Int64
	memoryFailureCount atomic.Int64
	memoryRetryCount   atomic.Int64
)

type MemoryMetricsSnapshot struct {
	QueueLength int
	Success     int64
	Failure     int64
	Retry       int64
}

func memoryQueueLength() int {
	if memoryTaskQueue == nil {
		return 0
	}
	return len(memoryTaskQueue)
}

func memoryMetricsSnapshot() MemoryMetricsSnapshot {
	return MemoryMetricsSnapshot{
		QueueLength: memoryQueueLength(),
		Success:     memorySuccessCount.Load(),
		Failure:     memoryFailureCount.Load(),
		Retry:       memoryRetryCount.Load(),
	}
}

func memoryMetricsLogKV() string {
	s := memoryMetricsSnapshot()
	return fmt.Sprintf("queue_len=%d success=%d failure=%d retry=%d", s.QueueLength, s.Success, s.Failure, s.Retry)
}
