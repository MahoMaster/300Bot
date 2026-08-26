package chatctx

import (
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// resetForTest 清空全局窗口并复位配置，保证用例互不干扰
func resetForTest() {
	windowsMu.Lock()
	windows = make(map[string]*window)
	windowsMu.Unlock()
	atomic.StoreInt64(&lastSweepUnix, 0)
	cfg = ctxConfig{
		windowSize:  50,
		maxChars:    4000,
		idleMinutes: 30,
		botQQ:       "10001",
		botName:     "叁柏",
	}
}

func TestAppendTrimsToWindowSize(t *testing.T) {
	resetForTest()
	cfg.windowSize = 3
	now := time.Now().Unix()
	for i := 1; i <= 5; i++ {
		AppendGroup("100", idStr(i), "200", "昵称", "消息"+idStr(i), now)
	}
	if got := WindowLen("100"); got != 3 {
		t.Fatalf("窗口条数 = %d, 期望 3", got)
	}
	rendered := SnapshotRendered("100")
	// 最旧的两条被裁掉
	if strings.Contains(rendered, "消息1") || strings.Contains(rendered, "消息2") {
		t.Fatalf("裁剪未生效: %s", rendered)
	}
	if !strings.Contains(rendered, "消息5") {
		t.Fatalf("最新消息丢失: %s", rendered)
	}
}

func TestDedupByMessageId(t *testing.T) {
	resetForTest()
	now := time.Now().Unix()
	AppendGroup("100", "m1", "200", "昵称", "同一条消息", now)
	AppendGroup("100", "m1", "200", "昵称", "同一条消息", now)
	if got := WindowLen("100"); got != 1 {
		t.Fatalf("message_id 去重失败, 条数 = %d", got)
	}
}

func TestSanitizeCQCode(t *testing.T) {
	resetForTest()
	now := time.Now().Unix()
	AppendGroup("100", "m1", "200", "昵称", "看这个[CQ:image,file=a.jpg,url=http://x][CQ:at,qq=1]好看", now)
	rendered := SnapshotRendered("100")
	if !strings.Contains(rendered, "[图片]") {
		t.Fatalf("图片 CQ 码未转占位: %s", rendered)
	}
	if strings.Contains(rendered, "CQ:") {
		t.Fatalf("CQ 码未剥离干净: %s", rendered)
	}
}

func TestEmptyAfterSanitizeDropped(t *testing.T) {
	resetForTest()
	now := time.Now().Unix()
	AppendGroup("100", "m1", "200", "昵称", "[CQ:face,id=176]", now)
	if got := WindowLen("100"); got != 0 {
		t.Fatalf("纯 CQ 码消息应丢弃, 条数 = %d", got)
	}
}

func TestSnapshotIdleFilter(t *testing.T) {
	resetForTest()
	now := time.Now().Unix()
	AppendGroup("100", "m1", "200", "昵称", "很久以前", now-3600) // 超出 30 分钟
	AppendGroup("100", "m2", "201", "昵称", "刚刚", now)
	rendered := SnapshotRendered("100")
	if strings.Contains(rendered, "很久以前") {
		t.Fatalf("空闲超时消息未被过滤: %s", rendered)
	}
	if !strings.Contains(rendered, "刚刚") {
		t.Fatalf("窗口内消息丢失: %s", rendered)
	}
}

func TestSnapshotCharBudget(t *testing.T) {
	resetForTest()
	cfg.maxChars = 60
	now := time.Now().Unix()
	for i := 1; i <= 5; i++ {
		AppendGroup("100", idStr(i), "200", "昵称", strings.Repeat("字", 30), now+int64(i))
	}
	rendered := SnapshotRendered("100")
	if len([]rune(rendered)) > 60 {
		t.Fatalf("字符预算未生效, 长度 = %d", len([]rune(rendered)))
	}
	// 预算从新到旧分配，最新一条必须在
	if !strings.Contains(rendered, "字") {
		t.Fatalf("快照为空")
	}
}

func TestSnapshotExcludeMessageIds(t *testing.T) {
	resetForTest()
	now := time.Now().Unix()
	AppendGroup("100", "m1", "200", "昵称", "闲聊内容", now)
	AppendGroup("100", "m2", "200", "昵称", "触发内容", now+1)
	rendered := SnapshotRendered("100", "m2")
	if strings.Contains(rendered, "触发内容") {
		t.Fatalf("排除项未生效: %s", rendered)
	}
	if !strings.Contains(rendered, "闲聊内容") {
		t.Fatalf("其余消息丢失: %s", rendered)
	}
}

func TestRenderDualIdentity(t *testing.T) {
	resetForTest()
	now := time.Now().Unix()
	AppendGroup("100", "m1", "200", "小明", "你好", now)
	AppendGroup("100", "m2", "201", "", "在吗", now+1)
	AppendBotReply("100", "我在")
	rendered := SnapshotRendered("100")
	if !strings.Contains(rendered, "用户[小明](QQ:200): 你好") {
		t.Fatalf("双身份格式错误: %s", rendered)
	}
	if !strings.Contains(rendered, "用户(QQ:201): 在吗") {
		t.Fatalf("无昵称格式错误: %s", rendered)
	}
	if !strings.Contains(rendered, "叁柏: 我在") {
		t.Fatalf("机器人回复渲染错误: %s", rendered)
	}
}

func TestPrependEntriesDedupAndOrder(t *testing.T) {
	resetForTest()
	now := time.Now().Unix()
	AppendGroup("100", "m5", "200", "昵称", "新消息", now)
	PrependEntries("100", []IngestEntry{
		{MessageId: "m1", Entry: Entry{QQ: "200", Nickname: "昵称", Text: "旧消息1", Time: now - 100}},
		{MessageId: "m5", Entry: Entry{QQ: "200", Nickname: "昵称", Text: "重复消息", Time: now}},
		{MessageId: "m2", Entry: Entry{QQ: "201", Nickname: "昵称", Text: "旧消息2", Time: now - 50}},
	})
	rendered := SnapshotRendered("100")
	if strings.Contains(rendered, "重复消息") {
		t.Fatalf("补拉去重失败: %s", rendered)
	}
	idx1 := strings.Index(rendered, "旧消息1")
	idx2 := strings.Index(rendered, "旧消息2")
	idx5 := strings.Index(rendered, "新消息")
	if idx1 < 0 || idx2 < 0 || idx5 < 0 || !(idx1 < idx2 && idx2 < idx5) {
		t.Fatalf("补拉顺序错误: %s", rendered)
	}
}

func TestSenderNickname(t *testing.T) {
	msg := map[string]interface{}{
		"sender": map[string]interface{}{
			"nickname": "网名",
			"card":     "群名片",
		},
	}
	if got := SenderNickname(msg); got != "群名片" {
		t.Fatalf("应优先群名片, got=%s", got)
	}
	msg["sender"] = map[string]interface{}{"nickname": "网名", "card": ""}
	if got := SenderNickname(msg); got != "网名" {
		t.Fatalf("名片为空应回退昵称, got=%s", got)
	}
	if got := SenderNickname(map[string]interface{}{}); got != "" {
		t.Fatalf("无 sender 应返回空, got=%s", got)
	}
}

func idStr(i int) string {
	return strconv.Itoa(i)
}
