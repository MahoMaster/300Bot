package memory

// MemorySummary 表示经过总结后的单条长期记忆候选。
// 结构化改造新增字段一律 omitempty：保护 fallback.go 存量回灌行的反序列化，只加不改不删。
type MemorySummary struct {
	Scope      string   `json:"scope"`
	UserId     string   `json:"user_id"`
	GroupId    string   `json:"group_id"`
	SessionId  string   `json:"session_id"`
	MessageId  string   `json:"message_id"`
	Source     string   `json:"source"`
	Text       string   `json:"text"`
	Summary    string   `json:"summary"`
	Tags       []string `json:"tags"`
	Importance int      `json:"importance"`
	Confidence float64  `json:"confidence"`
	CreatedAt  int64    `json:"created_at"`

	// 条目型记忆（memoryEntryExtractEnabled 开启后产出）；为空时按 legacy 路径处理，行为与改造前完全一致
	SubjectId     string `json:"subject_id,omitempty"`    // 记忆主体 QQ 号（群维度下可与 owner 群号不同）
	Type          string `json:"mem_type,omitempty"`      // identity|profile|preference|dislike|trait|relationship|habit|goal|rule|group_meme
	Key           string `json:"mem_key,omitempty"`       // 属性键，如 current_location/alias/game_dbd
	Evidence      string `json:"evidence,omitempty"`      // 支持该记忆的原文依据（仅存档，不参与 embedding）
	EvidenceCount int    `json:"evidence_count,omitempty"` // 累积证据次数，重复验证时由 Manager 递增
}

// IsEntry 是否为条目型记忆（具备结构化三元组）；legacy 记忆恒为 false
func (s MemorySummary) IsEntry() bool {
	return s.SubjectId != "" && s.Type != "" && s.Key != ""
}

