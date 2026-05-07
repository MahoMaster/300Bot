package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type MemoryRawTurn struct {
	Id        int64  `db:"id" json:"id"`
	Scope     string `db:"scope" json:"scope"`
	UserId    string `db:"user_id" json:"user_id"`
	GroupId   string `db:"group_id" json:"group_id"`
	SessionId string `db:"session_id" json:"session_id"`
	MessageId string `db:"message_id" json:"message_id"`
	Source    string `db:"source" json:"source"`
	InputText string `db:"input_text" json:"input_text"`
	ReplyText string `db:"reply_text" json:"reply_text"`
	Status    string `db:"status" json:"status"`
	CreatedAt int64  `db:"created_at" json:"created_at"`
	UpdatedAt int64  `db:"updated_at" json:"updated_at"`
}

type MemoryPendingOwnerStat struct {
	Scope        string `db:"scope" json:"scope"`
	OwnerId      string `db:"owner_id" json:"owner_id"`
	TurnCount    int    `db:"turn_count" json:"turn_count"`
	CharCount    int    `db:"char_count" json:"char_count"`
	FirstTurnAt  int64  `db:"first_turn_at" json:"first_turn_at"`
	LatestTurnAt int64  `db:"latest_turn_at" json:"latest_turn_at"`
}

func ensureMemoryRawTurnsTable() error {
	_, e := db.Exec(`
CREATE TABLE IF NOT EXISTS memory_raw_turns (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  scope VARCHAR(16) NOT NULL,
  user_id VARCHAR(32) NOT NULL DEFAULT '',
  group_id VARCHAR(32) NOT NULL DEFAULT '',
  session_id VARCHAR(64) NOT NULL,
  message_id VARCHAR(64) NOT NULL,
  source VARCHAR(16) NOT NULL DEFAULT '',
  input_text MEDIUMTEXT,
  reply_text MEDIUMTEXT,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  INDEX idx_scope_session_status (scope, session_id, status),
  INDEX idx_scope_message (scope, message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`)
	return e
}

func InsertMemoryRawTurn(turn MemoryRawTurn) (int64, error) {
	now := time.Now().Unix()
	res, e := db.Exec(
		"insert into memory_raw_turns (`scope`,`user_id`,`group_id`,`session_id`,`message_id`,`source`,`input_text`,`reply_text`,`status`,`created_at`,`updated_at`) values (?,?,?,?,?,?,?,?,?,?,?)",
		turn.Scope,
		turn.UserId,
		turn.GroupId,
		turn.SessionId,
		turn.MessageId,
		turn.Source,
		turn.InputText,
		turn.ReplyText,
		"pending",
		now,
		now,
	)
	if e != nil {
		return 0, e
	}
	id, e := res.LastInsertId()
	if e != nil {
		return 0, e
	}
	return id, nil
}

func UpsertMemoryRawOutput(turn MemoryRawTurn) error {
	now := time.Now().Unix()
	res, e := db.Exec(
		"update memory_raw_turns set reply_text=?,updated_at=? where scope=? and session_id=? and message_id=? order by id desc limit 1",
		turn.ReplyText,
		now,
		turn.Scope,
		turn.SessionId,
		turn.MessageId,
	)
	if e != nil {
		return e
	}
	aff, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if aff > 0 {
		return nil
	}
	_, e = InsertMemoryRawTurn(MemoryRawTurn{
		Scope:     turn.Scope,
		UserId:    turn.UserId,
		GroupId:   turn.GroupId,
		SessionId: turn.SessionId,
		MessageId: turn.MessageId,
		Source:    turn.Source,
		InputText: "",
		ReplyText: turn.ReplyText,
	})
	return e
}

func MarkMemoryRawTurnsStatus(ids []int64, status string) error {
	if len(ids) == 0 {
		return nil
	}
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("status 不能为空")
	}
	query, args, e := sqlx.In("update memory_raw_turns set status=?,updated_at=? where id in (?)", status, time.Now().Unix(), ids)
	if e != nil {
		return e
	}
	query = db.Rebind(query)
	_, e = db.Exec(query, args...)
	return e
}

func GetPendingMemoryRawTurns(scope string, sessionId string, limit int) []MemoryRawTurn {
	if limit <= 0 {
		limit = 100
	}
	var turns = make([]MemoryRawTurn, 0)
	e := db.Select(&turns, "select * from memory_raw_turns where scope=? and session_id=? and status='pending' order by id asc limit ?", scope, sessionId, limit)
	if e != nil {
		fmt.Println(e)
	}
	return turns
}

func GetPendingMemoryRawTurnsByOwner(scope string, ownerId string, limit int) []MemoryRawTurn {
	if limit <= 0 {
		limit = 100
	}
	scope = strings.TrimSpace(scope)
	ownerId = strings.TrimSpace(ownerId)
	if scope == "" || ownerId == "" {
		return make([]MemoryRawTurn, 0)
	}
	ownerColumn := ""
	switch scope {
	case "user":
		ownerColumn = "user_id"
	case "group":
		ownerColumn = "group_id"
	default:
		return make([]MemoryRawTurn, 0)
	}

	var turns = make([]MemoryRawTurn, 0)
	query := fmt.Sprintf("select * from memory_raw_turns where scope=? and %s=? and status='pending' order by id asc limit ?", ownerColumn)
	e := db.Select(&turns, query, scope, ownerId, limit)
	if e != nil {
		fmt.Println(e)
	}
	return turns
}

func ListPendingMemoryOwnerStats(limit int) []MemoryPendingOwnerStat {
	if limit <= 0 {
		limit = 200
	}
	var stats = make([]MemoryPendingOwnerStat, 0)
	e := db.Select(
		&stats,
		`select
			scope,
			case when scope='user' then user_id else group_id end as owner_id,
			count(1) as turn_count,
			sum(char_length(coalesce(input_text, '')) + char_length(coalesce(reply_text, ''))) as char_count,
			min(created_at) as first_turn_at,
			max(created_at) as latest_turn_at
		from memory_raw_turns
		where status='pending'
		group by scope, owner_id
		order by first_turn_at asc
		limit ?`,
		limit,
	)
	if e != nil {
		fmt.Println(e)
	}
	return stats
}
