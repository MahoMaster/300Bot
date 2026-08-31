package memory

import (
	"strings"
	"testing"
)

// Memory Manager 裁决相关纯函数测试（解析/置信度演化/目标点校验），不依赖 conf 与任何 IO。

func TestParseReconcileDecision(t *testing.T) {
	raw := "裁决结果如下：\n{\"decision\":\" UPDATE \",\"target_point_id\":\" abc-123 \",\"merged_value\":\" 搬到上海 \"}"
	decision, err := parseReconcileDecision(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if decision.Decision != "update" || decision.TargetPointId != "abc-123" || decision.MergedValue != "搬到上海" {
		t.Errorf("清洗不符: %+v", decision)
	}
	// confidence 越界夹取
	decision, err = parseReconcileDecision(`{"decision":"add","confidence":3.7}`)
	if err != nil || decision.Confidence != 1 {
		t.Errorf("confidence 应夹取到 1: %+v err=%v", decision, err)
	}
	// 无 JSON 应报错（调用方降级 ADD）
	if _, err := parseReconcileDecision("无法判断"); err == nil {
		t.Error("无 JSON 应返回错误")
	}
}

func TestBumpMemoryConfidence(t *testing.T) {
	// 旧点无置信度：起步值
	if got := bumpMemoryConfidence(0); got != memoryConfidenceFloor {
		t.Errorf("缺省起步应为 %v, got %v", memoryConfidenceFloor, got)
	}
	// 正常步进
	if got := bumpMemoryConfidence(0.7); got != 0.75 {
		t.Errorf("0.7+0.05 应为 0.75, got %v", got)
	}
	// 封顶
	if got := bumpMemoryConfidence(memoryConfidenceCap); got != memoryConfidenceCap {
		t.Errorf("封顶 %v, got %v", memoryConfidenceCap, got)
	}
}

func TestFindRecordById(t *testing.T) {
	records := []MemoryPointRecord{
		{Id: "p1", Payload: map[string]interface{}{"mem_key": "alias"}},
		{Id: "p2", Payload: map[string]interface{}{"mem_key": "current_location"}},
	}
	if rec := findRecordById(records, "p2"); rec == nil || rec.Id != "p2" {
		t.Error("应命中 p2")
	}
	// 编造 ID / 空 ID 必须落空（防误改无关点）
	if findRecordById(records, "p-fake") != nil || findRecordById(records, "  ") != nil {
		t.Error("不存在/空 ID 应返回 nil")
	}
}

func TestBuildAdjudicationUserPrompt(t *testing.T) {
	entry := MemorySummary{SubjectId: "675559614", Type: "profile", Key: "current_location", Summary: "搬到上海", Importance: 4, Confidence: 0.8}
	olds := []MemoryPointRecord{{
		Id: "p1",
		Payload: map[string]interface{}{
			"mem_type": "profile", "mem_key": "current_location",
			"text": "长期居住在成都", "confidence": 0.75, "evidence_count": float64(2),
		},
	}}
	prompt := buildAdjudicationUserPrompt(entry, olds)
	for _, want := range []string{"current_location", "搬到上海", "[ID: p1]", "长期居住在成都"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("裁决输入缺少 %q: %s", want, prompt)
		}
	}
}
