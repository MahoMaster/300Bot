package model

import (
	"log"
	"time"
)

// MemoryFallbackRow Qdrant 写入重试耗尽后暂存的总结记录（P9 MySQL fallback）
type MemoryFallbackRow struct {
	Id          int64  `db:"id" json:"id"`
	SummaryJSON string `db:"summary_json" json:"summary_json"`
	Status      string `db:"status" json:"status"`
	CreatedAt   int64  `db:"created_at" json:"created_at"`
	UpdatedAt   int64  `db:"updated_at" json:"updated_at"`
}

func ensureMemoryFallbackTable() error {
	_, e := db.Exec(`
CREATE TABLE IF NOT EXISTS memory_summary_fallback (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  summary_json MEDIUMTEXT NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`)
	return e
}

func InsertMemoryFallback(summaryJSON string) error {
	now := time.Now().Unix()
	_, e := db.Exec(
		"insert into memory_summary_fallback (`summary_json`,`status`,`created_at`,`updated_at`) values (?,?,?,?)",
		summaryJSON,
		"pending",
		now,
		now,
	)
	return e
}

func ListMemoryFallback(limit int) []MemoryFallbackRow {
	if limit <= 0 {
		limit = 20
	}
	var rows = make([]MemoryFallbackRow, 0)
	e := db.Select(&rows, "select * from memory_summary_fallback where status='pending' order by id asc limit ?", limit)
	if e != nil {
		log.Printf("memory fallback list failed err=%v", e)
	}
	return rows
}

func DeleteMemoryFallback(id int64) error {
	_, e := db.Exec("delete from memory_summary_fallback where id=?", id)
	return e
}
