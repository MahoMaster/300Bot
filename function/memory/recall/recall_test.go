package recall

import (
	"strings"
	"testing"
)

func TestParseSearchResponseNormal(t *testing.T) {
	body := []byte(`{"result":[{"score":0.87,"payload":{"text":"用户喜欢猫","summary":"偏好"}},{"score":0.5,"payload":{"text":"","summary":" 只有摘要 "}}]}`)
	hits, err := ParseSearchResponse(body, "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("want 2 hits, got %d", len(hits))
	}
	if hits[0].Score != 0.87 || hits[0].Text != "用户喜欢猫" || hits[0].Summary != "偏好" || hits[0].Scope != "user" {
		t.Errorf("hit[0] mismatch: %+v", hits[0])
	}
	if hits[1].Text != "" || hits[1].Summary != "只有摘要" {
		t.Errorf("hit[1] trim/summary mismatch: %+v", hits[1])
	}
}

func TestParseSearchResponseEmptyResult(t *testing.T) {
	hits, err := ParseSearchResponse([]byte(`{"result":[]}`), "group")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("want empty hits, got %d", len(hits))
	}
}

func TestParseSearchResponseInvalidJSON(t *testing.T) {
	if _, err := ParseSearchResponse([]byte(`not-json`), "user"); err == nil {
		t.Fatal("want error for invalid json")
	}
}

func TestMergeHitsThresholdAndTopK(t *testing.T) {
	userHits := []MemoryHit{
		{Score: 0.9, Text: "u1"},
		{Score: 0.6, Text: "u2"},
		{Score: 0.2, Text: "u-low"}, // 低于阈值
	}
	groupHits := []MemoryHit{
		{Score: 0.8, Text: "g1"},
		{Score: 0.7, Text: "g2"},
		{Score: 0.5, Text: "g3"},
	}
	merged := MergeHits(userHits, groupHits, 2, 0.35)
	// 每路最多 topK=2：user 取 u1,u2；group 取 g1,g2；u-low 被阈值过滤
	if len(merged) != 4 {
		t.Fatalf("want 4 hits, got %d: %+v", len(merged), merged)
	}
	// 分数降序
	scores := make([]float64, 0, len(merged))
	for _, h := range merged {
		scores = append(scores, h.Score)
		if h.Text == "u-low" || h.Text == "g3" {
			t.Errorf("hit %s should be filtered", h.Text)
		}
	}
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[i-1] {
			t.Errorf("scores not descending: %v", scores)
		}
	}
}

func TestMergeHitsDedup(t *testing.T) {
	userHits := []MemoryHit{{Score: 0.9, Text: "相同内容"}}
	groupHits := []MemoryHit{{Score: 0.8, Text: " 相同内容 "}}
	merged := MergeHits(userHits, groupHits, 4, 0.35)
	if len(merged) != 1 {
		t.Fatalf("want 1 hit after dedup, got %d: %+v", len(merged), merged)
	}
	if merged[0].Score != 0.9 {
		t.Errorf("want higher-score copy kept, got %+v", merged[0])
	}
}

func TestMergeHitsEmptyBoth(t *testing.T) {
	if merged := MergeHits(nil, nil, 4, 0.35); len(merged) != 0 {
		t.Errorf("want empty, got %+v", merged)
	}
	// Text 与 Summary 均为空的命中应被丢弃
	merged := MergeHits([]MemoryHit{{Score: 0.9, Text: "  "}}, nil, 4, 0.35)
	if len(merged) != 0 {
		t.Errorf("want blank hit dropped, got %+v", merged)
	}
}

func TestRenderTextHeaderAndFormat(t *testing.T) {
	hits := []MemoryHit{
		{Score: 0.87, Text: "用户喜欢猫"},
		{Score: 0.6, Text: "", Summary: "养过狗"},
	}
	text := RenderText(hits, 2000)
	if !strings.HasPrefix(text, "【关于对方的既有记忆】") {
		t.Errorf("missing header: %s", text)
	}
	if !strings.Contains(text, "- (相似度0.87) 用户喜欢猫") {
		t.Errorf("text line mismatch: %s", text)
	}
	if !strings.Contains(text, "- (相似度0.60) 养过狗") {
		t.Errorf("summary fallback mismatch: %s", text)
	}
}

func TestRenderTextEmpty(t *testing.T) {
	if text := RenderText(nil, 2000); text != "" {
		t.Errorf("want empty for no hits, got %q", text)
	}
	if text := RenderText([]MemoryHit{{Score: 0.9, Text: "x"}}, 0); text != "" {
		t.Errorf("want empty for non-positive budget, got %q", text)
	}
}

func TestRenderTextRuneBudget(t *testing.T) {
	// 头部占 10 rune + 换行 1，预算 20 时剩余 9 rune，不够放任何一行
	hits := []MemoryHit{{Score: 0.9, Text: "很长的记忆内容文本"}}
	if text := RenderText(hits, 20); text != "" {
		t.Errorf("want empty when budget cannot fit any line, got %q", text)
	}
	// 预算足够放一行、不够两行
	hits2 := []MemoryHit{
		{Score: 0.9, Text: "甲"},
		{Score: 0.8, Text: "乙"},
	}
	text := RenderText(hits2, 26)
	if !strings.Contains(text, "甲") || strings.Contains(text, "乙") {
		t.Errorf("budget truncation mismatch: %q", text)
	}
	if len([]rune(text)) > 26 {
		t.Errorf("rendered text exceeds budget: %d runes", len([]rune(text)))
	}
}

func TestPreviewText(t *testing.T) {
	if got := PreviewText("短文本", 80); got != "短文本" {
		t.Errorf("short text should be unchanged, got %q", got)
	}
	long := strings.Repeat("字", 100)
	got := PreviewText(long, 80)
	if len([]rune(got)) != 83 || !strings.HasSuffix(got, "...") {
		t.Errorf("long text preview mismatch: %d runes", len([]rune(got)))
	}
}
