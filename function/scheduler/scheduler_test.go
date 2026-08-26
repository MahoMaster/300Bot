package scheduler

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 同一 key 的任务必须按提交顺序串行执行
func TestFIFOOrderPerKey(t *testing.T) {
	s := New("test-fifo", 4, 200, time.Minute)
	total := 100
	order := make([]int, 0, total)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < total; i++ {
		i := i
		wg.Add(1)
		if !s.Submit("k1", func() {
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			wg.Done()
		}) {
			t.Fatalf("submit %d dropped unexpectedly", i)
		}
	}
	wg.Wait()

	if len(order) != total {
		t.Fatalf("expected %d executed jobs, got %d", total, len(order))
	}
	for i, v := range order {
		if v != i {
			t.Fatalf("execution order broken at index %d: %v", i, order)
		}
	}
}

// 队列打满后 Submit 必须返回 false（丢弃）而不是阻塞
func TestDropWhenQueueFull(t *testing.T) {
	s := New("test-drop", 1, 2, time.Minute)
	block := make(chan struct{})
	started := make(chan struct{})

	if !s.Submit("k", func() {
		close(started)
		<-block
	}) {
		t.Fatal("first submit dropped unexpectedly")
	}
	<-started // 确保首个任务正在执行，后续提交只进队列

	if !s.Submit("k", func() {}) {
		t.Fatal("second submit dropped unexpectedly")
	}
	if !s.Submit("k", func() {}) {
		t.Fatal("third submit dropped unexpectedly")
	}
	if s.Submit("k", func() {}) {
		t.Fatal("expected submit to be dropped when queue is full")
	}
	close(block)
}

// 不同 key 的任务应当并发执行
func TestParallelAcrossKeys(t *testing.T) {
	s := New("test-parallel", 4, 8, time.Minute)
	jobs := 3
	var running int64
	block := make(chan struct{})
	startedAll := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < jobs; i++ {
		key := strconv.Itoa(i)
		wg.Add(1)
		if !s.Submit(key, func() {
			if atomic.AddInt64(&running, 1) == int64(jobs) {
				close(startedAll)
			}
			<-block
			atomic.AddInt64(&running, -1)
			wg.Done()
		}) {
			t.Fatalf("submit key=%s dropped unexpectedly", key)
		}
	}

	select {
	case <-startedAll:
	case <-time.After(3 * time.Second):
		close(block)
		t.Fatalf("jobs did not run concurrently, running=%d", atomic.LoadInt64(&running))
	}
	close(block)
	wg.Wait()
}

// 任务 panic 不应影响调度器后续任务执行
func TestJobPanicRecovered(t *testing.T) {
	s := New("test-panic", 1, 4, time.Minute)
	done := make(chan struct{})
	s.Submit("k", func() {
		panic("boom")
	})
	if !s.Submit("k", func() {
		close(done)
	}) {
		t.Fatal("second submit dropped unexpectedly")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("scheduler stopped working after job panic")
	}
}
