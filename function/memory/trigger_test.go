package memory

import "testing"

// 事件触发纯函数测试：关键词预筛与 per-owner 冷却，不依赖 conf 与任何 IO。

func TestMatchEvent(t *testing.T) {
	keywords := []string{"以后叫我", "记住", "我搬到", "不再是"}
	cases := []struct {
		text string
		want bool
	}{
		{"以后叫我王哥", true},
		{"我搬到上海了", true},
		{"今天天气不错", false},
		{"   ", false},
	}
	for _, c := range cases {
		if got := MatchEvent(c.text, keywords); got != c.want {
			t.Errorf("MatchEvent(%q)=%v, 期望 %v", c.text, got, c.want)
		}
	}
	// 空词表/全空白词不命中（防配置缺失导致全量触发）
	if MatchEvent("记住这句话", nil) || MatchEvent("记住这句话", []string{"", "  "}) {
		t.Error("空词表/空白词不应命中")
	}
}

func TestEventTriggerAllowed(t *testing.T) {
	// 冷却关闭时恒允许
	if !eventTriggerAllowed("owner-nocd", 0) || !eventTriggerAllowed("owner-nocd", 0) {
		t.Error("cooldown<=0 应恒允许")
	}
	// 冷却期内只允许一次
	if !eventTriggerAllowed("owner-cd", 300) {
		t.Error("首次应允许")
	}
	if eventTriggerAllowed("owner-cd", 300) {
		t.Error("冷却期内应拒绝")
	}
	// 不同 owner 互不影响
	if !eventTriggerAllowed("owner-other", 300) {
		t.Error("其他 owner 不受冷却影响")
	}
}
