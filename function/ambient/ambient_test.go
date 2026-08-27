package ambient

import (
	"300Bot/function/chatctx"
	"sync/atomic"
	"testing"
	"time"
)

// seedWindow 预置两条群友消息入窗，满足最小窗口长度要求
func seedWindow(groupId string) {
	now := time.Now().Unix()
	chatctx.AppendGroup(groupId, groupId+"-m1", "200", "小明", "今天天气不错", now)
	chatctx.AppendGroup(groupId, groupId+"-m2", "201", "小红", "是啊挺舒服的", now)
}

// installCounter 安装计数回调并返回计数器
func installCounter() *int32 {
	var count int32
	SetDecideCallback(func(groupId string) {
		atomic.AddInt32(&count, 1)
	})
	return &count
}

// waitCount 轮询等待计数达到期望值，超时返回当时的计数
func waitCount(count *int32, want int32, timeout time.Duration) int32 {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if got := atomic.LoadInt32(count); got >= want {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	return atomic.LoadInt32(count)
}

func TestGateDisabledAndWhitelist(t *testing.T) {
	count := installCounter()
	// 总开关关闭
	Configure(false, []string{"100"}, 100, 600, 0, 0, "999")
	seedWindow("g-disabled")
	OnGroupMessage("g-disabled", "200", "闲聊一句")
	// 开关开但群不在白名单
	Configure(true, []string{"100"}, 100, 600, 0, 0, "999")
	seedWindow("g-outside")
	OnGroupMessage("g-outside", "200", "闲聊一句")
	if got := waitCount(count, 1, 200*time.Millisecond); got != 0 {
		t.Fatalf("关闭/白名单外不应触发回调, got=%d", got)
	}
}

func TestChanceZeroNeverTriggers(t *testing.T) {
	count := installCounter()
	Configure(true, []string{"g-zero"}, 0, 600, 0, 0, "999")
	seedWindow("g-zero")
	for i := 0; i < 20; i++ {
		OnGroupMessage("g-zero", "200", "刷屏消息")
	}
	if got := waitCount(count, 1, 200*time.Millisecond); got != 0 {
		t.Fatalf("chance=0 不应触发回调, got=%d", got)
	}
}

func TestChanceHundredTriggers(t *testing.T) {
	count := installCounter()
	Configure(true, []string{"g-full"}, 100, 600, 0, 0, "999")
	seedWindow("g-full")
	OnGroupMessage("g-full", "200", "闲聊一句")
	if got := waitCount(count, 1, 2*time.Second); got != 1 {
		t.Fatalf("chance=100 应触发一次回调, got=%d", got)
	}
}

func TestPendingDedup(t *testing.T) {
	count := installCounter()
	// 思考延迟拉长到 1 秒，保证第二条消息落在 pending 期内
	Configure(true, []string{"g-dedup"}, 100, 600, 1, 1, "999")
	seedWindow("g-dedup")
	OnGroupMessage("g-dedup", "200", "第一条")
	OnGroupMessage("g-dedup", "201", "第二条应被 pending 挡住")
	if got := waitCount(count, 1, 3*time.Second); got != 1 {
		t.Fatalf("pending 期内第二条应丢弃, 最终回调数 = %d", got)
	}
}

func TestCooldownAfterNotifyReplied(t *testing.T) {
	count := installCounter()
	Configure(true, []string{"g-cd"}, 100, 600, 0, 0, "999")
	seedWindow("g-cd")
	OnGroupMessage("g-cd", "200", "闲聊一句")
	if got := waitCount(count, 1, 2*time.Second); got != 1 {
		t.Fatalf("首次应触发回调, got=%d", got)
	}
	// 插话成功 → 刷新冷却 → 下一条消息应被冷却挡住
	NotifyReplied("g-cd")
	OnGroupMessage("g-cd", "201", "冷却期内的消息")
	if got := waitCount(count, 2, 300*time.Millisecond); got != 1 {
		t.Fatalf("冷却期内不应再次触发, got=%d", got)
	}
}

func TestPureCQAndBotSenderAndBotLastSkipped(t *testing.T) {
	count := installCounter()
	Configure(true, []string{"g-skip"}, 100, 600, 0, 0, "999")
	seedWindow("g-skip")
	// 纯 CQ 码消息
	OnGroupMessage("g-skip", "200", "[CQ:image,file=a.jpg,url=http://x]")
	// 发言人是机器人自己
	OnGroupMessage("g-skip", "999", "自己的消息")
	// 上一条是机器人发言（模拟 AppendBotReply 后立刻来新消息的场景需先清 pending，
	// 这里直接构造窗口末条为机器人）
	chatctx.AppendBotReply("g-skip", "机器人刚说完")
	OnGroupMessage("g-skip", "200", "接在机器人后面的消息")
	if got := waitCount(count, 1, 300*time.Millisecond); got != 0 {
		t.Fatalf("纯CQ/自己发言/机器人末条均不应触发, got=%d", got)
	}
}

func TestWindowTooShortSkipped(t *testing.T) {
	count := installCounter()
	Configure(true, []string{"g-short"}, 100, 600, 0, 0, "999")
	// 只入窗一条：当前消息本身入窗后仍不足最小窗口长度
	now := time.Now().Unix()
	chatctx.AppendGroup("g-short", "g-short-m1", "200", "小明", "只有这一条", now)
	OnGroupMessage("g-short", "200", "只有这一条")
	if got := waitCount(count, 1, 200*time.Millisecond); got != 0 {
		t.Fatalf("窗口太短不应触发, got=%d", got)
	}
}
