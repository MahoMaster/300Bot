// Package chatctx 维护群聊全量消息的滑动窗口，
// 为 AI 应答注入触发前的环境群聊记录（阶段二上下文重建）。
// 本包不依赖 conf/send，窗口参数通过 Configure 注入，便于单元测试。
package chatctx

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Entry 滑动窗口中的一条消息记录
type Entry struct {
	QQ       string
	Nickname string
	Text     string
	Time     int64
	IsBot    bool
	msgId    string // 用于去重，不参与外部渲染
}

// IngestEntry 补拉历史消息条目，带 message_id 用于与窗口内记录去重
type IngestEntry struct {
	MessageId string
	Entry
}

type window struct {
	mu         sync.Mutex
	entries    []Entry
	messageIds map[string]struct{}
	lastActive int64
}

type ctxConfig struct {
	windowSize  int
	maxChars    int
	idleMinutes int
	botQQ       string
	botName     string
}

var cfg = ctxConfig{
	windowSize:  50,
	maxChars:    4000,
	idleMinutes: 30,
	botQQ:       "",
	botName:     "bot",
}

var (
	windowsMu     sync.RWMutex
	windows       = make(map[string]*window)
	lastSweepUnix int64
	botReplySeq   int64
)

var (
	cqImageRe = regexp.MustCompile(`\[CQ:image[^\]]*\]`)
	cqCodeRe  = regexp.MustCompile(`\[CQ:[^\]]*\]`)
)

// Configure 覆盖窗口参数，小于等于 0 / 空串的项保持默认；应在启动时调用一次。
// 窗口本身按条数与字符预算有界，参数轻微竞态不影响正确性。
func Configure(windowSize, maxChars, idleMinutes int, botQQ, botName string) {
	if windowSize > 0 {
		cfg.windowSize = windowSize
	}
	if maxChars > 0 {
		cfg.maxChars = maxChars
	}
	if idleMinutes > 0 {
		cfg.idleMinutes = idleMinutes
	}
	if strings.TrimSpace(botQQ) != "" {
		cfg.botQQ = strings.TrimSpace(botQQ)
	}
	if strings.TrimSpace(botName) != "" {
		cfg.botName = strings.TrimSpace(botName)
	}
}

func groupKey(groupId string) string {
	return "group:" + groupId
}

func getWindow(groupId string) *window {
	windowsMu.RLock()
	w := windows[groupKey(groupId)]
	windowsMu.RUnlock()
	return w
}

func getOrCreateWindow(groupId string) *window {
	key := groupKey(groupId)
	windowsMu.RLock()
	w, ok := windows[key]
	windowsMu.RUnlock()
	if ok {
		return w
	}
	windowsMu.Lock()
	defer windowsMu.Unlock()
	sweepLocked(time.Now().Unix())
	if w, ok = windows[key]; ok {
		return w
	}
	w = &window{messageIds: make(map[string]struct{})}
	windows[key] = w
	return w
}

// sweepLocked 惰性回收长期无活动的窗口，调用方需持有 windowsMu 写锁，每分钟最多扫一次
func sweepLocked(now int64) {
	if now-atomic.LoadInt64(&lastSweepUnix) < 60 {
		return
	}
	atomic.StoreInt64(&lastSweepUnix, now)
	idleSec := int64(cfg.idleMinutes) * 60 * 2
	for k, w := range windows {
		w.mu.Lock()
		stale := w.lastActive > 0 && now-w.lastActive > idleSec
		w.mu.Unlock()
		if stale {
			delete(windows, k)
		}
	}
}

// sanitizeText 剥离 CQ 码：图片段转 [图片] 占位，其余 CQ 段直接移除
func sanitizeText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = cqImageRe.ReplaceAllString(text, "[图片]")
	text = cqCodeRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// trimLocked 按窗口容量裁掉最旧条目并同步清理去重表，调用方需持有 w.mu
func trimLocked(w *window) {
	for len(w.entries) > cfg.windowSize {
		drop := w.entries[0]
		w.entries = w.entries[1:]
		if drop.msgId != "" {
			delete(w.messageIds, drop.msgId)
		}
	}
}

// AppendGroup 追加一条群成员消息入窗；剥离 CQ 码后为空的消息直接丢弃，
// messageId 非空时按 message_id 去重
func AppendGroup(groupId, messageId, qq, nickname, text string, ts int64) {
	text = sanitizeText(text)
	if groupId == "" || text == "" {
		return
	}
	if ts <= 0 {
		ts = time.Now().Unix()
	}
	w := getOrCreateWindow(groupId)
	w.mu.Lock()
	defer w.mu.Unlock()
	if messageId != "" {
		if _, dup := w.messageIds[messageId]; dup {
			return
		}
		w.messageIds[messageId] = struct{}{}
	}
	w.entries = append(w.entries, Entry{
		QQ:       qq,
		Nickname: nickname,
		Text:     text,
		Time:     ts,
		msgId:    messageId,
	})
	trimLocked(w)
	w.lastActive = time.Now().Unix()
}

// AppendBotReply 追加机器人自己的回复入窗（NapCat 不回推机器人自己的群消息，需发送后手动记录）
func AppendBotReply(groupId, text string) {
	text = sanitizeText(text)
	if groupId == "" || text == "" {
		return
	}
	now := time.Now().Unix()
	msgId := fmt.Sprintf("bot:%d:%d", now, atomic.AddInt64(&botReplySeq, 1))
	w := getOrCreateWindow(groupId)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messageIds[msgId] = struct{}{}
	w.entries = append(w.entries, Entry{
		QQ:       cfg.botQQ,
		Nickname: cfg.botName,
		Text:     text,
		Time:     now,
		IsBot:    true,
		msgId:    msgId,
	})
	trimLocked(w)
	w.lastActive = now
}

// PrependEntries 将补拉的历史消息按调用方给定的时间升序插到窗口头部，按 message_id 去重
func PrependEntries(groupId string, items []IngestEntry) {
	if groupId == "" || len(items) == 0 {
		return
	}
	w := getOrCreateWindow(groupId)
	w.mu.Lock()
	defer w.mu.Unlock()
	fresh := make([]Entry, 0, len(items))
	for _, item := range items {
		text := sanitizeText(item.Text)
		if text == "" {
			continue
		}
		if item.MessageId != "" {
			if _, dup := w.messageIds[item.MessageId]; dup {
				continue
			}
			w.messageIds[item.MessageId] = struct{}{}
		}
		e := item.Entry
		e.Text = text
		e.msgId = item.MessageId
		fresh = append(fresh, e)
	}
	if len(fresh) == 0 {
		return
	}
	w.entries = append(fresh, w.entries...)
	trimLocked(w)
	if w.lastActive == 0 {
		w.lastActive = time.Now().Unix()
	}
}

// WindowLen 返回窗口当前条数，用于补拉的空洞判定
func WindowLen(groupId string) int {
	w := getWindow(groupId)
	if w == nil {
		return 0
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.entries)
}

// LastEntryIsBot 返回窗口最后一条是否机器人发言，用于自主插话闸门避免对着自己的话连环接话；
// 窗口不存在或为空返回 false
func LastEntryIsBot(groupId string) bool {
	w := getWindow(groupId)
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.entries) == 0 {
		return false
	}
	return w.entries[len(w.entries)-1].IsBot
}

// SnapshotRendered 将窗口内未超出空闲超时的消息渲染为群聊记录文本（时间升序），
// 从新到旧按字符预算截断；excludeMsgIds 用于排除即将单独作为触发消息出现的条目
func SnapshotRendered(groupId string, excludeMsgIds ...string) string {
	selected := selectSnapshot(groupId, excludeMsgIds...)
	if len(selected) == 0 {
		return ""
	}
	lines := make([]string, 0, len(selected))
	for _, e := range selected {
		lines = append(lines, renderLine(e))
	}
	return strings.Join(lines, "\n")
}

// snapshotEntry 结构化快照条目（阶段四 JSON 化输入）
type snapshotEntry struct {
	QQ       string `json:"qq"`
	Nickname string `json:"nickname"`
	Text     string `json:"text"`
	Time     int64  `json:"time"`
	Bot      bool   `json:"bot"`
}

// SnapshotJSON 与 SnapshotRendered 相同过滤规则，但输出 JSON 数组文本（time 为 unix 秒）；
// 空窗口或序列化失败返回 ""
func SnapshotJSON(groupId string, excludeMsgIds ...string) string {
	selected := selectSnapshot(groupId, excludeMsgIds...)
	if len(selected) == 0 {
		return ""
	}
	items := make([]snapshotEntry, 0, len(selected))
	for _, e := range selected {
		items = append(items, snapshotEntry{
			QQ:       e.QQ,
			Nickname: e.Nickname,
			Text:     e.Text,
			Time:     e.Time,
			Bot:      e.IsBot,
		})
	}
	body, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(body)
}

// selectSnapshot 复制窗口并按空闲超时/exclude/字符预算（从新到旧）过滤，返回时间升序候选
func selectSnapshot(groupId string, excludeMsgIds ...string) []Entry {
	w := getWindow(groupId)
	if w == nil {
		return nil
	}
	exclude := make(map[string]struct{}, len(excludeMsgIds))
	for _, id := range excludeMsgIds {
		if id != "" {
			exclude[id] = struct{}{}
		}
	}

	w.mu.Lock()
	entries := make([]Entry, len(w.entries))
	copy(entries, w.entries)
	w.mu.Unlock()

	cutoff := time.Now().Unix() - int64(cfg.idleMinutes)*60
	budget := cfg.maxChars
	selected := make([]Entry, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Time < cutoff {
			break
		}
		if _, skip := exclude[e.msgId]; skip {
			continue
		}
		cost := len([]rune(renderLine(e))) + 1
		if cost > budget {
			break
		}
		budget -= cost
		selected = append(selected, e)
	}
	// 反转为时间升序
	for i, j := 0, len(selected)-1; i < j; i, j = i+1, j-1 {
		selected[i], selected[j] = selected[j], selected[i]
	}
	return selected
}

// renderLine 双身份格式：群友以 昵称+QQ 号标注，机器人直呼其名
func renderLine(e Entry) string {
	if e.IsBot {
		return cfg.botName + ": " + e.Text
	}
	if e.Nickname != "" {
		return fmt.Sprintf("用户[%s](QQ:%s): %s", e.Nickname, e.QQ, e.Text)
	}
	return fmt.Sprintf("用户(QQ:%s): %s", e.QQ, e.Text)
}

// SenderNickname 从 OneBot 消息事件中安全提取发言人昵称（群名片优先）
func SenderNickname(msg map[string]interface{}) string {
	sender, ok := msg["sender"].(map[string]interface{})
	if !ok {
		return ""
	}
	if card, ok := sender["card"].(string); ok && strings.TrimSpace(card) != "" {
		return strings.TrimSpace(card)
	}
	if nickname, ok := sender["nickname"].(string); ok {
		return strings.TrimSpace(nickname)
	}
	return ""
}
