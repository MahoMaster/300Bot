package memory

import (
	"strings"
	"testing"
)

// 条目化提取的纯函数测试（解析/规范化/渲染），不依赖 conf 与任何 IO。

func TestParseEntryExtraction(t *testing.T) {
	raw := "```json\n{\"memories\":[{\"subject_id\":\"675559614\",\"type\":\"profile\",\"key\":\"current_location\",\"value\":\"长期居住在成都\",\"importance\":4,\"confidence\":0.8,\"evidence\":\"我住在成都好几年了\"}]}\n```"
	entries, err := parseEntryExtraction(raw)
	if err != nil {
		t.Fatalf("parseEntryExtraction 失败: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("期望 1 条, 实际 %d", len(entries))
	}
	if entries[0].SubjectId != "675559614" || entries[0].Key != "current_location" {
		t.Errorf("解析字段不符: %+v", entries[0])
	}
}

func TestParseEntryExtractionEmpty(t *testing.T) {
	// 闲聊批次常态：空数组是合法结果
	entries, err := parseEntryExtraction(`{"memories":[]}`)
	if err != nil {
		t.Fatalf("空数组应合法: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("期望 0 条, 实际 %d", len(entries))
	}
	// 无 JSON 内容应报错（由调用方标记失败留补扫）
	if _, err := parseEntryExtraction("完全没有 JSON 的闲聊回复"); err == nil {
		t.Error("无 JSON 应返回错误")
	}
}

func TestNormalizeMemoryEntryType(t *testing.T) {
	cases := map[string]string{
		"identity":     "identity",
		" Preference ": "preference",
		"GROUP_MEME":   "group_meme",
		"未知类型":     "profile",
		"":             "profile",
	}
	for input, want := range cases {
		if got := normalizeMemoryEntryType(input); got != want {
			t.Errorf("normalizeMemoryEntryType(%q)=%q, 期望 %q", input, got, want)
		}
	}
}

func TestSanitizeEntries(t *testing.T) {
	longValue := strings.Repeat("记", memoryEntryMaxValueRunes+50)
	entries := []MemoryEntry{
		{SubjectId: " 111 ", Type: "preference", Key: " Game_DBD ", Value: "喜欢玩DBD", Importance: 9, Confidence: 2.5},
		{SubjectId: "111", Type: "preference", Key: "game_dbd", Value: "喜欢玩DBD（重复三元组应去重）"},
		{SubjectId: "", Type: "profile", Key: "x", Value: "缺 subject 应丢弃"},
		{SubjectId: "222", Type: "profile", Key: "y", Value: ""},
		{SubjectId: "222", Type: "profile", Key: "z", Value: ""},
		{SubjectId: "333", Type: "trait", Key: "long", Value: longValue, Importance: 3, Confidence: 0.7},
	}
	cleaned := sanitizeEntries(entries)
	if len(cleaned) != 2 {
		t.Fatalf("期望 2 条（去重+丢弃非法）, 实际 %d: %+v", len(cleaned), cleaned)
	}
	first := cleaned[0]
	if first.SubjectId != "111" || first.Key != "game_dbd" || first.Type != "preference" {
		t.Errorf("清洗结果不符: %+v", first)
	}
	if first.Importance != 5 || first.Confidence != 1 {
		t.Errorf("importance/confidence 未夹取: %d / %v", first.Importance, first.Confidence)
	}
	if got := len([]rune(cleaned[1].Value)); got != memoryEntryMaxValueRunes {
		t.Errorf("value 未截断到 %d: got %d", memoryEntryMaxValueRunes, got)
	}
	if sanitizeEntries(nil) != nil {
		t.Error("空输入应返回 nil")
	}
}

func TestSanitizeEntriesBatchLimit(t *testing.T) {
	entries := make([]MemoryEntry, 0, memoryEntryMaxPerBatch+3)
	for i := 0; i < memoryEntryMaxPerBatch+3; i++ {
		entries = append(entries, MemoryEntry{SubjectId: "111", Type: "profile", Key: string(rune('a' + i)), Value: "v"})
	}
	if got := sanitizeEntries(entries); len(got) != memoryEntryMaxPerBatch {
		t.Errorf("每批上限 %d, 实际 %d", memoryEntryMaxPerBatch, len(got))
	}
}

func TestRenderEntryText(t *testing.T) {
	// value 不含 QQ 号：前缀拼接
	got := renderEntryText(MemoryEntry{SubjectId: "675559614", Value: "长期居住在成都"})
	if got != "QQ:675559614 长期居住在成都" {
		t.Errorf("渲染不符: %s", got)
	}
	// value 已含 QQ 号：不重复拼接
	got = renderEntryText(MemoryEntry{SubjectId: "675559614", Value: "QQ:675559614 长期居住在成都"})
	if got != "QQ:675559614 长期居住在成都" {
		t.Errorf("含 QQ 号不应重复拼接: %s", got)
	}
}

func TestBuildEntryDedupKey(t *testing.T) {
	// 同三元组稳定同键（文本如何变化都不影响）
	a := BuildEntryDedupKey("user", "owner", "675559614", "profile", "current_location")
	b := BuildEntryDedupKey(" USER ", " owner ", "675559614", "Profile", " Current_Location ")
	if a != b {
		t.Error("同三元组（忽略大小写/空白）应同键")
	}
	// 不同 key 必须不同键
	c := BuildEntryDedupKey("user", "owner", "675559614", "profile", "job")
	if a == c {
		t.Error("不同 key 应不同键")
	}
	// 与旧文本哈希键互不冲突
	if BuildDedupKey("user", "owner", "text") == a {
		t.Error("条目键不应与文本键碰撞")
	}
}
