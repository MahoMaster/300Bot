package message

import (
	"300Bot/function/repeat"
	"300Bot/send"
	"300Bot/store"
	"fmt"
	"strconv"
	"strings"

	"github.com/thinkeridea/go-extend/exstrings"
)

func CheckType(msg map[string]interface{}) {
	switch msg["message_type"] {
	case "private":
		private(msg)
		break
	case "group":
		group(msg)
		break
	default:
		break
	}
}

// var groupIdList []float64

//私聊消息
func private(msg map[string]interface{}) {
	fmt.Println("私聊消息", msg["raw_message"])
	send.SendPrivate(msg["user_id"].(float64), msg["raw_message"].(string))
}

//群消息
func group(msg map[string]interface{}) {
	fmt.Println("群消息", msg)
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

	//查询at
	self_id_str := strconv.FormatFloat(self_id, 'f', -1, 64)
	if strings.Index(msg["raw_message"].(string), "[CQ:at,qq="+self_id_str+"]") != -1 {
		msgStr = exstrings.Replace(msgStr, "[CQ:at,qq="+self_id_str+"]", "", -1)
		msgStr = strings.TrimSpace(msgStr)
		//如果是at的关键词就直接结束
		if checkAtWords(msgStr, msg) {
			return
		}
	}
	msgStr = strings.TrimSpace(msgStr)
	//获取关键字
	msgArr := strings.Fields(msgStr)
	if len(msgArr) > 0 {
		if checkKeywords(msgArr[0], msgStr, msg) {
			return
		}

	}

	// fmt.Println(self_id)
	repeat.CheckRepeat(msg)
}
