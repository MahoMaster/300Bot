package memory

import (
	"300Bot/conf"
	"300Bot/model"
	"encoding/json"
	"log"
)

// MySQL fallback（P9）：Qdrant 写入重试耗尽后落 memory_summary_fallback 表，
// 定时回灌恢复后重新写入 Qdrant

// SaveFallback 将总结记录序列化后暂存 MySQL；失败仅记日志（L1 兜底已是最后防线）
func SaveFallback(summary MemorySummary) {
	data, err := json.Marshal(summary)
	if err != nil {
		log.Printf("memory fallback marshal failed scope=%s err=%v", summary.Scope, err)
		return
	}
	if err := model.InsertMemoryFallback(string(data)); err != nil {
		log.Printf("memory fallback insert failed scope=%s user=%s group=%s err=%v", summary.Scope, summary.UserId, summary.GroupId, err)
		return
	}
	log.Printf("memory fallback saved scope=%s user=%s group=%s", summary.Scope, summary.UserId, summary.GroupId)
}

// BackfillFallback 定时回灌：取最多 20 条暂存记录串行重写 Qdrant（复用 dedup 窗口检查），
// 单条失败即停本轮（服务大概率仍不可用，等下轮），避免雪崩
func BackfillFallback() {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryFallbackToMysql {
		return
	}
	rows := model.ListMemoryFallback(20)
	backfilled := 0
	for _, row := range rows {
		var summary MemorySummary
		if err := json.Unmarshal([]byte(row.SummaryJSON), &summary); err != nil {
			log.Printf("memory fallback unmarshal failed id=%d err=%v (drop row)", row.Id, err)
			if delErr := model.DeleteMemoryFallback(row.Id); delErr != nil {
				log.Printf("memory fallback delete failed id=%d err=%v", row.Id, delErr)
			}
			backfilled++
			continue
		}
		if _, err := UpsertMemorySummary(summary); err != nil {
			log.Printf("memory fallback backfill stopped backfilled=%d remaining>=%d err=%v", backfilled, len(rows)-backfilled, err)
			return
		}
		if err := model.DeleteMemoryFallback(row.Id); err != nil {
			log.Printf("memory fallback delete failed id=%d err=%v", row.Id, err)
			return
		}
		backfilled++
	}
	if len(rows) > 0 {
		log.Printf("memory fallback backfill done backfilled=%d", backfilled)
	}
}
