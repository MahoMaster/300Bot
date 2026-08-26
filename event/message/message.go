package message

import (
	"300Bot/conf"
	"300Bot/function/bangDream/station"
	"300Bot/function/chatGPT"
	"300Bot/function/chatctx"
	"300Bot/function/emotion"
	"300Bot/function/immortal"
	memoryCollector "300Bot/function/memory"
	"300Bot/function/repeat"
	"300Bot/store"
	"log"
	"strconv"
	"strings"

	"github.com/thinkeridea/go-extend/exstrings"
)

func CheckType(msg map[string]interface{}) {
	switch msg["message_type"] {
	case "private":
		private(msg)

	case "group":
		group(msg)

	default:

	}
}

// var groupIdList []float64

// 私聊消息
func private(msg map[string]interface{}) {
	log.Println("私聊消息", msg["raw_message"])
	session := chatGPT.ResolveSession(msg, 1)
	if session == "" {
		session = strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	}
	memoryCollector.CollectInput("user", "private", session, msg)
	// send.SendPrivate(msg["user_id"].(float64), `[CQ:image,file=0c9df9e9aaa98350bb28c1ca2661c5e0.image]`)
	// go func() {
	// time.Sleep(1 * time.Second)
	chatGPT.AddPlanPrivate(msg["raw_message"].(string), msg)
	// send.SendPrivate(msg["user_id"].(float64), "有什么意见建议问题可以直接发给Maho")
	// }()

}

// 群消息
func group(msg map[string]interface{}) {
	log.Println("群消息", msg)
	//是否被ban
	banIndex := -1
	for key, value := range store.BanList {
		if value.Qq == strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64) {
			banIndex = key
		}
	}
	if banIndex != -1 {
		return
	}

	if msg["sub_type"] != "normal" {
		return
	}
	//是否在服务的群内
	// if arrays.ContainsFloat(groupIdList, msg["group_id"].(float64)) == -1 {
	// 	return
	// }
	groupIndex := -1
	for key, value := range store.GroupList {
		if value.Group_id == msg["group_id"].(float64) {
			groupIndex = key
		}
	}
	if groupIndex == -1 {
		return
	}
	self_id := msg["self_id"].(float64)

	msgStr := msg["raw_message"].(string)
	session := chatGPT.ResolveSession(msg, 0)
	if session == "" {
		session = strconv.FormatFloat(msg["group_id"].(float64), 'f', -1, 64)
	}
	memoryCollector.CollectInput("group", "group", session, msg)
	appendChatWindow(msg)

	//查询#号，接入修仙游戏
	msgStr = strings.TrimSpace(msgStr)
	if len(msgStr) != 0 && msgStr[0] == '#' {
		if immortal.CheckKeywords(msgStr, msg) {
			return
		}
	}

	//查询at
	self_id_str := strconv.FormatFloat(self_id, 'f', -1, 64)
	if strings.Contains(msg["raw_message"].(string), "[CQ:at,qq="+self_id_str+"]") {
		msgStr = exstrings.Replace(msgStr, "[CQ:at,qq="+self_id_str+"]", "", -1)
		msgStr = strings.TrimSpace(msgStr)
		//如果是at的关键词就直接结束
		if checkAtWords(msgStr, msg) {
			return
		}
	}

	//查询呼叫机器人名字
	botName := conf.Config.BotName
	if strings.Contains(msg["raw_message"].(string), botName) {
		msgStr = strings.TrimSpace(msgStr)

		chatGPT.AddPlan(msgStr, msg)
		//呼叫chatGPT,结束
		return

	}

	msgStr = strings.TrimSpace(msgStr)
	//获取关键字
	msgArr := strings.Fields(msgStr)
	if len(msgArr) > 0 {
		if checkKeywords(msgArr[0], msgStr, msg) {
			return
		}

	}

	//获取结尾
	if len(msgStr) >= 4 && msgStr[len(msgStr)-4:] == ".jpg" {
		msgArr = strings.Split(msgStr, ".jpg")
		if len(msgArr) >= 2 {
			emotion.Synthesis(msgArr[0], msg)
		}
		return
	}

	//检查车牌
	if station.CheckSubmitRoom(msgStr, msg) {
		return
	}

	// fmt.Println(self_id)
	repeat.CheckRepeat(msg)
}

// appendChatWindow 将全量群聊（含未触发机器人的普通消息）追加进滑动窗口，
// AI 应答的环境上下文以此为准；非服务群/ban 用户已在上游过滤
func appendChatWindow(msg map[string]interface{}) {
	groupId, ok := msg["group_id"].(float64)
	if !ok {
		return
	}
	userId, _ := msg["user_id"].(float64)
	rawText, _ := msg["raw_message"].(string)
	var ts int64
	if t, ok := msg["time"].(float64); ok {
		ts = int64(t)
	}
	msgId := ""
	if id, ok := msg["message_id"].(float64); ok {
		msgId = strconv.FormatFloat(id, 'f', -1, 64)
	}
	chatctx.AppendGroup(
		strconv.FormatFloat(groupId, 'f', -1, 64),
		msgId,
		strconv.FormatFloat(userId, 'f', -1, 64),
		chatctx.SenderNickname(msg),
		rawText,
		ts,
	)
}
