package memory

// MemorySummary 表示经过总结后的单条长期记忆候选。
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
}

