package memory

import "testing"

// 生命周期衰减判定纯函数测试，不依赖 conf 与任何 IO。
// payload 数值一律 float64（与 JSON 反序列化结果一致）。

func decayRecord(mutate func(map[string]interface{})) MemoryPointRecord {
	payload := map[string]interface{}{
		"status":         memoryStatusActive,
		"updated_at":     float64(1_000_000),
		"importance":     float64(4),
		"confidence":     0.7,
		"evidence_count": float64(1),
	}
	if mutate != nil {
		mutate(payload)
	}
	return MemoryPointRecord{Id: "p1", Payload: payload}
}

func TestDecideDecayActionFresh(t *testing.T) {
	// 更新不久（>= 过期线）→ 不动；注意 staleCutoff 需早于 updated_at 才算"近期"
	now := int64(1_100_000)
	staleCutoff := int64(999_000)
	if action, _ := decideDecayAction(decayRecord(nil), now, staleCutoff); action != "" {
		t.Errorf("近期更新不应衰减, got %q", action)
	}
	// 非 active（已被软删/过期）→ 不动
	rec := decayRecord(func(p map[string]interface{}) { p["status"] = memoryStatusDeleted })
	if action, _ := decideDecayAction(rec, now, int64(0)); action != "" {
		t.Errorf("非 active 不应再处理, got %q", action)
	}
}

func TestDecideDecayActionValidUntil(t *testing.T) {
	// goal 类显式有效期到期 → 直接过期（即使刚更新）
	rec := decayRecord(func(p map[string]interface{}) {
		p["valid_until"] = float64(1_000)
		p["updated_at"] = float64(999_999)
	})
	action, kv := decideDecayAction(rec, 2_000, 1)
	if action != memoryStatusExpired || kv["status"] != memoryStatusExpired {
		t.Errorf("valid_until 到期应过期, got %q %v", action, kv)
	}
}

func TestDecideDecayActionStale(t *testing.T) {
	now := int64(2_000_000)
	staleCutoff := int64(1_500_000)

	// 陈旧 + 低重要度 → 直接过期
	rec := decayRecord(func(p map[string]interface{}) { p["importance"] = float64(2) })
	if action, _ := decideDecayAction(rec, now, staleCutoff); action != memoryStatusExpired {
		t.Errorf("低重要度陈旧点应过期, got %q", action)
	}

	// 陈旧 + 高重要度 + 置信度尚高 → 降置信度
	action, kv := decideDecayAction(decayRecord(nil), now, staleCutoff)
	if action != "weaken" {
		t.Fatalf("高价值陈旧点应先降置信度, got %q", action)
	}
	if conf, ok := kv["confidence"].(float64); !ok || conf < 0.6-1e-9 || conf > 0.7 {
		t.Errorf("置信度应降到 0.6, got %v", kv["confidence"])
	}

	// 陈旧 + 高重要度 + 置信度已到下限 → 过期
	rec = decayRecord(func(p map[string]interface{}) { p["confidence"] = memoryDecayConfidenceFloor })
	if action, _ := decideDecayAction(rec, now, staleCutoff); action != memoryStatusExpired {
		t.Errorf("置信度到下限应过期, got %q", action)
	}

	// updated_at 缺失回退 created_at
	rec = decayRecord(func(p map[string]interface{}) {
		delete(p, "updated_at")
		p["created_at"] = float64(1_000_000)
	})
	if action, _ := decideDecayAction(rec, now, staleCutoff); action != "weaken" {
		t.Errorf("应回退 created_at 判定, got %q", action)
	}

	// updated_at 与 created_at 均缺失 → 保守不动
	rec = decayRecord(func(p map[string]interface{}) { delete(p, "updated_at") })
	if action, _ := decideDecayAction(rec, now, staleCutoff); action != "" {
		t.Errorf("无时间戳应保守不动, got %q", action)
	}
}
