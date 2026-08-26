package inline

import (
	"strings"
	"testing"
)

func TestParseReplyStandardJSON(t *testing.T) {
	raw := `{"should_reply":true,"reply":"你好呀","memory":["用户(QQ:200)喜欢猫"]}`
	got := ParseReply(raw)
	if !got.ShouldReply || got.Reply != "你好呀" || len(got.Memory) != 1 || got.Memory[0] != "用户(QQ:200)喜欢猫" {
		t.Fatalf("标准 JSON 解析错误: %+v", got)
	}
}

func TestParseReplyMarkdownFenced(t *testing.T) {
	raw := "```json\n{\"should_reply\":true,\"reply\":\"带围栏\",\"memory\":[]}\n```"
	got := ParseReply(raw)
	if got.Reply != "带围栏" {
		t.Fatalf("markdown 围栏解析错误: %+v", got)
	}
}

func TestParseReplySurroundingText(t *testing.T) {
	raw := "好的，这是结果：{\"should_reply\":true,\"reply\":\"正文\"} 以上。"
	got := ParseReply(raw)
	if got.Reply != "正文" {
		t.Fatalf("前后杂文本解析错误: %+v", got)
	}
}

func TestParseReplyInvalidJSONFallback(t *testing.T) {
	// 含大括号但非法 JSON：整段当 reply 兜底
	raw := "这不是{合法JSON"
	got := ParseReply(raw)
	if !got.ShouldReply || got.Reply != raw {
		t.Fatalf("非法 JSON 应整段兜底为 reply: %+v", got)
	}
	// 无大括号的纯文本：同样整段当 reply
	raw2 := "今天天气不错"
	got2 := ParseReply(raw2)
	if !got2.ShouldReply || got2.Reply != raw2 || got2.Memory != nil {
		t.Fatalf("纯文本应整段兜底为 reply: %+v", got2)
	}
}

func TestParseReplyEmpty(t *testing.T) {
	got := ParseReply("   ")
	if got.ShouldReply || got.Reply != "" || got.Memory != nil {
		t.Fatalf("空输入应返回零值: %+v", got)
	}
}

func TestParseReplyShouldReplyFalseKept(t *testing.T) {
	raw := `{"should_reply":false,"reply":"","memory":[]}`
	got := ParseReply(raw)
	if got.ShouldReply || got.Reply != "" {
		t.Fatalf("should_reply=false 应保留: %+v", got)
	}
}

func TestNormalizeReplyTrimAndDropEmpty(t *testing.T) {
	r := NormalizeReply(ChatReply{
		Reply:  "  回复  ",
		Memory: []string{"  有效候选  ", "", "   ",},
	})
	if r.Reply != "回复" {
		t.Fatalf("reply trim 失败: %q", r.Reply)
	}
	if len(r.Memory) != 1 || r.Memory[0] != "有效候选" {
		t.Fatalf("空串候选未丢弃: %+v", r.Memory)
	}
}

func TestNormalizeReplyRuneTruncate(t *testing.T) {
	long := strings.Repeat("记", MaxCandidateRunes+100)
	r := NormalizeReply(ChatReply{Memory: []string{long}})
	if len(r.Memory) != 1 || len([]rune(r.Memory[0])) != MaxCandidateRunes {
		t.Fatalf("候选未按 %d rune 截断: %d", MaxCandidateRunes, len([]rune(r.Memory[0])))
	}
}

func TestNormalizeReplyMaxCountAndDedup(t *testing.T) {
	memory := make([]string, 0, MaxCandidates+3)
	memory = append(memory, "重复候选", "重复候选")
	for i := 0; i < MaxCandidates+1; i++ {
		memory = append(memory, "候选"+string(rune('A'+i)))
	}
	r := NormalizeReply(ChatReply{Memory: memory})
	if len(r.Memory) != MaxCandidates {
		t.Fatalf("候选数未按上限 %d 截断: %d", MaxCandidates, len(r.Memory))
	}
	seen := make(map[string]struct{}, len(r.Memory))
	for _, item := range r.Memory {
		if _, dup := seen[item]; dup {
			t.Fatalf("去重失败: %+v", r.Memory)
		}
		seen[item] = struct{}{}
	}
}

func TestNormalizeReplyAllEmptyBecomesNil(t *testing.T) {
	r := NormalizeReply(ChatReply{Memory: []string{"", "  "}})
	if r.Memory != nil {
		t.Fatalf("全空候选应归 nil: %+v", r.Memory)
	}
}
