// Package ambient 自主插话（环境回复）本地闸门：
// 所有显式触发都未命中的群消息在此做白名单/概率/冷却/去重等零成本判定，
// 通过后再经随机"思考延迟"触发 LLM 决策回调。
// 本包不依赖 conf/send/chatGPT，参数经 Configure 注入、决策逻辑经回调注入，便于单元测试。
package ambient

import (
	"300Bot/function/chatctx"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"
)

// minWindowLen 窗口内至少要有这么多条消息才值得考虑插话
// （当前消息本身已入窗，为 1 说明刚见过这个群，没有环境上下文）
const minWindowLen = 2

type ambientConfig struct {
	enabled                         bool
	whitelistOn                     bool
	groups                          map[string]struct{}
	chance, cooldownSec             int
	thinkMinSec, thinkMaxSec        int
	botQQ                           string
}

// groupState 每群的插话状态：冷却截止时间与待决定时器标记
type groupState struct {
	nextAllowedUnix int64
	pending         bool
}

var (
	stateMu        sync.Mutex
	cfg            ambientConfig
	states         = make(map[string]*groupState)
	decideCallback func(groupId string)
)

// cqCodeRe 剔除 CQ 码段，与 chatctx.sanitizeText 语义对齐的简化版（独立实现避免跨包引私有函数）
var cqCodeRe = regexp.MustCompile(`\[CQ:[^\]]*\]`)

// Configure 注入闸门参数，应在启动时调用一次；非法/空值由调用方（配置层）先补默认。
// whitelistOn 为 false 时忽略 groups 白名单，所有群都可进入后续闸门判定
func Configure(enabled, whitelistOn bool, groups []string, chance, cooldownSec, thinkMinSec, thinkMaxSec int, botQQ string) {
	g := make(map[string]struct{}, len(groups))
	for _, id := range groups {
		id = strings.TrimSpace(id)
		if id != "" {
			g[id] = struct{}{}
		}
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	cfg = ambientConfig{
		enabled:      enabled,
		whitelistOn:  whitelistOn,
		groups:       g,
		chance:       chance,
		cooldownSec:  cooldownSec,
		thinkMinSec:  thinkMinSec,
		thinkMaxSec:  thinkMaxSec,
		botQQ:        botQQ,
	}
}

// SetDecideCallback 注入 LLM 决策回调（由 chatGPT 包提供）；未注入时 OnGroupMessage 为 no-op
func SetDecideCallback(fn func(groupId string)) {
	stateMu.Lock()
	defer stateMu.Unlock()
	decideCallback = fn
}

// NotifyReplied 插话发送成功后刷新冷却，避免下一条消息立刻再过闸
func NotifyReplied(groupId string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	st := getOrCreateStateLocked(groupId)
	st.nextAllowedUnix = time.Now().Unix() + int64(cfg.cooldownSec)
}

// OnGroupMessage 自主插话闸门入口：仅对未命中任何显式触发的群消息调用。
// 全部判定为内存操作；通过后才启动思考延迟定时器，到期触发 LLM 决策回调
func OnGroupMessage(groupId, userId, rawText string) {
	if groupId == "" {
		return
	}
	// 空消息/纯 CQ 码没有插话价值
	if strings.TrimSpace(cqCodeRe.ReplaceAllString(rawText, "")) == "" {
		return
	}

	stateMu.Lock()
	if !cfg.enabled {
		stateMu.Unlock()
		return
	}
	if _, ok := cfg.groups[groupId]; cfg.whitelistOn && !ok {
		stateMu.Unlock()
		return
	}
	// 发言人是机器人自己（NapCat 本就不回推自己消息，双保险）
	if userId != "" && userId == cfg.botQQ {
		stateMu.Unlock()
		return
	}
	chance := cfg.chance
	thinkMin := cfg.thinkMinSec
	thinkMax := cfg.thinkMaxSec
	stateMu.Unlock()

	// 上一条是机器人自己的发言则跳过，避免对着自己的话连环插话
	if chatctx.LastEntryIsBot(groupId) {
		return
	}
	// 窗口太短没有环境上下文可判断
	if chatctx.WindowLen(groupId) < minWindowLen {
		return
	}

	now := time.Now().Unix()
	stateMu.Lock()
	st := getOrCreateStateLocked(groupId)
	// 冷却未到 / 该群已有待决定时器 → 丢弃
	if st.pending || now < st.nextAllowedUnix {
		stateMu.Unlock()
		return
	}
	// 概率骰子：chance 为放行百分比（不引 util 包：util 间接依赖 conf，会破坏本包单测独立性）
	if randInt(1, 100) > chance {
		stateMu.Unlock()
		return
	}
	st.pending = true
	stateMu.Unlock()

	// 思考延迟：模拟人类反应节奏，到期时拿到的窗口快照也更新鲜
	delay := time.Duration(randInt(thinkMin, thinkMax)) * time.Second
	time.AfterFunc(delay, func() {
		stateMu.Lock()
		if cur, ok := states[groupId]; ok {
			cur.pending = false
		}
		fn := decideCallback
		stateMu.Unlock()
		if fn != nil {
			fn(groupId)
		}
	})
}

// getOrCreateStateLocked 调用方需已持有 stateMu
func getOrCreateStateLocked(groupId string) *groupState {
	st, ok := states[groupId]
	if !ok {
		st = &groupState{}
		states[groupId] = st
	}
	return st
}

// randInt 返回 [start, end] 闭区间随机整数；Go 1.20+ 全局源自动随机种子，
// 独立实现以免引入 util 包的 conf 依赖（同 chatctx 不引 conf/send 的惯例）
func randInt(start, end int) int {
	if end < start {
		return start
	}
	return rand.Intn(end-start+1) + start
}
