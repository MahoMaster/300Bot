package memory

import (
	"300Bot/conf"
	"300Bot/model"
	"log"
	"time"
)

// 定时补扫与清理（P8/P10/P14），由 interval.go cron 调用

// ScanPendingOwners 扫描 pending/failed 状态的 owner，至多触发 50 个重新总结；
// 阈值判断（条数/字符/超时）由 TryBatchSummarizeOwner 内部完成，不满足直接返回
func ScanPendingOwners() {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryBatchEnabled {
		return
	}
	stats := model.ListPendingMemoryOwnerStats(200)
	if len(stats) == 0 {
		return
	}
	triggered := 0
	for _, stat := range stats {
		if triggered >= 50 {
			break
		}
		switch stat.Scope {
		case "user":
			go TryBatchSummarizeOwner("user", stat.OwnerId, "")
		case "group":
			go TryBatchSummarizeOwner("group", "", stat.OwnerId)
		default:
			continue
		}
		triggered++
	}
	log.Printf("memory scan pending owners=%d triggered=%d", len(stats), triggered)
}

// CleanupSummarizedTurns 分批清理超过保留天数的已总结记录（每批 5000 防长事务）
func CleanupSummarizedTurns() {
	if !conf.Memory.MemoryEnabled {
		return
	}
	days := conf.Memory.MemoryRawRetentionDays
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().Unix() - int64(days)*86400
	total := int64(0)
	for {
		affected, err := model.DeleteSummarizedMemoryRawTurns(cutoff, 5000)
		if err != nil {
			log.Printf("memory cleanup failed deleted=%d err=%v", total, err)
			return
		}
		total += affected
		if affected == 0 {
			break
		}
	}
	if total > 0 {
		log.Printf("memory cleanup done retention_days=%d deleted=%d", days, total)
	}
}
