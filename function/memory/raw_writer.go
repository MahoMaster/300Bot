package memory

import (
	"300Bot/conf"
	"300Bot/model"
	"log"
	"sync"
	"time"
)

// CollectInput 异步批量写入器（P13）：
// 消息热路径只非阻塞入队，单 worker 攒批或定时 flush，主链路零 DB 阻塞

var (
	rawTurnQueue      chan model.MemoryRawTurn
	rawWriterInitOnce sync.Once
)

func initRawWriter() {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryRawStoreEnabled {
		return
	}
	rawWriterInitOnce.Do(func() {
		rawTurnQueue = make(chan model.MemoryRawTurn, conf.Memory.MemoryRawQueueSize)
		go rawWriterLoop()
		log.Printf("memory raw writer started queue=%d batch=%d", conf.Memory.MemoryRawQueueSize, conf.Memory.MemoryRawBatchSize)
	})
}

// enqueueRawTurn 非阻塞入队；队列满仅记日志丢弃，绝不阻塞消息管道
func enqueueRawTurn(turn model.MemoryRawTurn) bool {
	initRawWriter()
	if rawTurnQueue == nil {
		return false
	}
	select {
	case rawTurnQueue <- turn:
		return true
	default:
		log.Printf("memory raw enqueue dropped scope=%s session=%s message=%s err=queue_full", turn.Scope, turn.SessionId, turn.MessageId)
		return false
	}
}

func rawWriterLoop() {
	batchSize := conf.Memory.MemoryRawBatchSize
	if batchSize <= 0 {
		batchSize = 20
	}
	pending := make([]model.MemoryRawTurn, 0, batchSize)
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(time.Second)
	}
	for {
		select {
		case turn := <-rawTurnQueue:
			pending = append(pending, turn)
			if len(pending) >= batchSize {
				flushRawTurns(pending, batchSize)
				pending = make([]model.MemoryRawTurn, 0, batchSize)
				resetTimer()
			}
		case <-timer.C:
			if len(pending) > 0 {
				flushRawTurns(pending, batchSize)
				pending = make([]model.MemoryRawTurn, 0, batchSize)
			}
			timer.Reset(time.Second)
		}
	}
}

// flushRawTurns 分批批量插入；成功后对批内各 turn 触发总结（总结触发时机后移到落库后）
func flushRawTurns(turns []model.MemoryRawTurn, batchSize int) {
	for _, batch := range splitBatches(turns, batchSize) {
		if err := model.BatchInsertMemoryRawTurns(batch); err != nil {
			log.Printf("memory raw batch insert failed count=%d err=%v", len(batch), err)
			continue
		}
		for _, turn := range batch {
			go TryBatchSummarizeOwner(turn.Scope, turn.UserId, turn.GroupId)
		}
	}
}

// splitBatches 按 size 切片；size<=0 时整批返回，空切片返回 nil
func splitBatches(turns []model.MemoryRawTurn, size int) [][]model.MemoryRawTurn {
	if len(turns) == 0 {
		return nil
	}
	if size <= 0 || size >= len(turns) {
		return [][]model.MemoryRawTurn{turns}
	}
	batches := make([][]model.MemoryRawTurn, 0, (len(turns)+size-1)/size)
	for start := 0; start < len(turns); start += size {
		end := start + size
		if end > len(turns) {
			end = len(turns)
		}
		batches = append(batches, turns[start:end])
	}
	return batches
}
