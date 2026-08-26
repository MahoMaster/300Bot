package scheduler

import (
	"log"
	"sync"
	"time"
)

// Scheduler 按 key 维度串行执行任务（同 key FIFO），不同 key 之间并行，
// 并通过信号量限制全局同时在途的任务数。入队为非阻塞：队列满时丢弃任务并返回 false。
// 会话空闲超过 idleTimeout 后自动回收对应 worker。
type Scheduler struct {
	name        string
	sem         chan struct{} // 全局并发信号量，容量 = maxRunning
	queueDepth  int
	idleTimeout time.Duration

	mu     sync.Mutex
	queues map[string]chan func()
}

// New 创建调度器。maxRunning 为全局并发上限，queueDepth 为每个 key 的队列深度，
// idleTimeout 为 worker 空闲回收时间。非法参数会回落到安全默认值。
func New(name string, maxRunning, queueDepth int, idleTimeout time.Duration) *Scheduler {
	if maxRunning <= 0 {
		maxRunning = 1
	}
	if queueDepth <= 0 {
		queueDepth = 1
	}
	if idleTimeout <= 0 {
		idleTimeout = 10 * time.Minute
	}
	return &Scheduler{
		name:        name,
		sem:         make(chan struct{}, maxRunning),
		queueDepth:  queueDepth,
		idleTimeout: idleTimeout,
		queues:      make(map[string]chan func()),
	}
}

// Submit 非阻塞提交任务到 key 对应的 FIFO 队列。
// 队列已满时返回 false（任务被丢弃），调用方可据此做兜底提示。
func (s *Scheduler) Submit(key string, fn func()) bool {
	if fn == nil {
		return false
	}
	// 入队全程持锁：与 worker 空闲退出的 delete 操作互斥，保证不丢任务
	s.mu.Lock()
	defer s.mu.Unlock()
	q, ok := s.queues[key]
	if !ok {
		q = make(chan func(), s.queueDepth)
		s.queues[key] = q
		go s.worker(key, q)
	}
	select {
	case q <- fn:
		return true
	default:
		log.Printf("scheduler %s: queue full, job dropped key=%s", s.name, key)
		return false
	}
}

// worker 串行消费某个 key 的队列；获取全局槽位时阻塞只影响本 key。
func (s *Scheduler) worker(key string, q chan func()) {
	timer := time.NewTimer(s.idleTimeout)
	defer timer.Stop()
	for {
		select {
		case fn := <-q:
			s.sem <- struct{}{}
			s.safeRun(key, fn)
			<-s.sem
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(s.idleTimeout)
		case <-timer.C:
			// 空闲退出：必须持锁确认队列为空再删除，
			// 与 Submit 的持锁入队互斥，避免"删队列瞬间入队"丢任务
			s.mu.Lock()
			if len(q) == 0 {
				delete(s.queues, key)
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
			timer.Reset(s.idleTimeout)
		}
	}
}

func (s *Scheduler) safeRun(key string, fn func()) {
	defer func() {
		if info := recover(); info != nil {
			log.Printf("scheduler %s: job panic key=%s info=%v", s.name, key, info)
		}
	}()
	fn()
}
