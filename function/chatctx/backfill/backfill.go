// Package backfill 在聊天窗口空洞（重启/崩溃/漏收）时从 NapCat 一次性补拉历史消息。
// 补拉全程异步 fire-and-forget，绝不阻塞触发路径；拉不到（本地缓存过期）自然降级为空。
package backfill

import (
	"300Bot/conf"
	"300Bot/function/chatctx"
	"300Bot/send"
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// backfillCooldownSec 同一群两次补拉的最小间隔，避免反复打 NapCat
	backfillCooldownSec = 300
	// backfillMinEntries 窗口条数达到该值视为无空洞，不补拉
	backfillMinEntries = 3
	// memberCacheTTLSec 群成员昵称缓存有效期
	memberCacheTTLSec = 1800
)

var (
	backfillLast sync.Map // groupId -> 上次补拉时间戳 int64
	memberCache  sync.Map // groupId:qq -> *memberCacheItem
)

type memberCacheItem struct {
	nickname string
	expireAt int64
}

// EnsureGroupWindow 窗口空洞且不在冷却期内时，异步补拉一次群历史
func EnsureGroupWindow(groupId string) {
	if groupId == "" {
		return
	}
	if chatctx.WindowLen(groupId) >= backfillMinEntries {
		return
	}
	now := time.Now().Unix()
	if v, ok := backfillLast.Load(groupId); ok {
		if last, ok := v.(int64); ok && now-last < backfillCooldownSec {
			return
		}
	}
	backfillLast.Store(groupId, now)
	go backfillGroup(groupId)
}

func backfillGroup(groupId string) {
	defer func() {
		if info := recover(); info != nil {
			log.Printf("chatctx backfill panic group=%s info=%v", groupId, info)
		}
	}()
	gid, err := strconv.ParseFloat(groupId, 64)
	if err != nil {
		return
	}
	res := send.GetGroupMsgHistory(gid, conf.Config.CtxBackfillCount)
	items := parseHistoryMessages(res)
	if len(items) == 0 {
		log.Printf("chatctx backfill empty group=%s", groupId)
		return
	}
	refreshNicknames(groupId, items)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Time < items[j].Time })
	chatctx.PrependEntries(groupId, items)
	log.Printf("chatctx backfill done group=%s count=%d", groupId, len(items))
}

// parseHistoryMessages 解析 NapCat 历史消息响应，兼容 raw_message 字符串与 message 分段数组两种形态
func parseHistoryMessages(raw []byte) []chatctx.IngestEntry {
	if len(raw) == 0 {
		return nil
	}
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Messages []map[string]interface{} `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		log.Printf("chatctx backfill parse failed: %v", err)
		return nil
	}
	if resp.Status != "ok" {
		return nil
	}
	items := make([]chatctx.IngestEntry, 0, len(resp.Data.Messages))
	for _, m := range resp.Data.Messages {
		text := extractMessageText(m)
		if strings.TrimSpace(text) == "" {
			continue
		}
		qq, nickname := "", ""
		if sender, ok := m["sender"].(map[string]interface{}); ok {
			qq = idString(sender["user_id"])
			if card, ok := sender["card"].(string); ok && strings.TrimSpace(card) != "" {
				nickname = strings.TrimSpace(card)
			} else if n, ok := sender["nickname"].(string); ok {
				nickname = strings.TrimSpace(n)
			}
		}
		items = append(items, chatctx.IngestEntry{
			MessageId: idString(m["message_id"]),
			Entry: chatctx.Entry{
				QQ:       qq,
				Nickname: nickname,
				Text:     text,
				Time:     int64Field(m["time"]),
				IsBot:    qq != "" && qq == conf.Config.BotQQ,
			},
		})
	}
	return items
}

// extractMessageText 从单条历史消息提取文本：优先 raw_message，否则扁平化 message 分段
func extractMessageText(m map[string]interface{}) string {
	if raw, ok := m["raw_message"].(string); ok && strings.TrimSpace(raw) != "" {
		return raw
	}
	segs, ok := m["message"].([]interface{})
	if !ok {
		return ""
	}
	var sb strings.Builder
	for _, segAny := range segs {
		seg, ok := segAny.(map[string]interface{})
		if !ok {
			continue
		}
		segType, _ := seg["type"].(string)
		data, _ := seg["data"].(map[string]interface{})
		switch segType {
		case "text":
			if t, ok := data["text"].(string); ok {
				sb.WriteString(t)
			}
		case "image":
			sb.WriteString("[图片]")
		case "record":
			sb.WriteString("[语音]")
		case "video":
			sb.WriteString("[视频]")
		}
	}
	return sb.String()
}

// refreshNicknames 用群成员信息刷新补拉消息的昵称（历史消息自带昵称可能已过期），
// 同一发言人只查一次；API 失败沿用消息自带昵称
func refreshNicknames(groupId string, items []chatctx.IngestEntry) {
	seen := make(map[string]string) // qq -> 刷新后的昵称
	for i := range items {
		if items[i].IsBot || items[i].QQ == "" {
			continue
		}
		nickname, ok := seen[items[i].QQ]
		if !ok {
			nickname = resolveNickname(groupId, items[i].QQ, items[i].Nickname)
			seen[items[i].QQ] = nickname
		}
		if nickname != "" {
			items[i].Nickname = nickname
		}
	}
}

func resolveNickname(groupId, qq, fallback string) string {
	key := groupId + ":" + qq
	now := time.Now().Unix()
	if v, ok := memberCache.Load(key); ok {
		if item, ok := v.(*memberCacheItem); ok && now < item.expireAt && item.nickname != "" {
			return item.nickname
		}
	}
	gid, err1 := strconv.ParseFloat(groupId, 64)
	uid, err2 := strconv.ParseFloat(qq, 64)
	if err1 != nil || err2 != nil {
		return fallback
	}
	nickname := parseMemberNickname(send.GetGroupMemberInfo(gid, uid))
	if nickname == "" {
		nickname = fallback
	}
	memberCache.Store(key, &memberCacheItem{nickname: nickname, expireAt: now + memberCacheTTLSec})
	return nickname
}

func parseMemberNickname(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Card     string `json:"card"`
			Nickname string `json:"nickname"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || resp.Status != "ok" {
		return ""
	}
	if card := strings.TrimSpace(resp.Data.Card); card != "" {
		return card
	}
	return strings.TrimSpace(resp.Data.Nickname)
}

func idString(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

func int64Field(raw interface{}) int64 {
	switch v := raw.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}
