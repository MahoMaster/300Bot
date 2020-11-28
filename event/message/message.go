package message

import (
	"300Bot/function/emotion"
	"300Bot/function/repeat"
	"300Bot/send"
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
	log.Println("私聊消息", msg["raw_message"])
	// send.SendPrivate(msg["user_id"].(float64), `[CQ:image,file=0c9df9e9aaa98350bb28c1ca2661c5e0.image]`)
	// go func() {
	// time.Sleep(1 * time.Second)
	send.SendPrivate(msg["user_id"].(float64), "有什么意见建议问题可以直接发给Maho")
	// }()

}

//群消息
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

	//获取结尾
	if len(msgStr) >= 4 && msgStr[len(msgStr)-4:] == ".jpg" {
		msgArr = strings.Split(msgStr, ".jpg")
		if len(msgArr) >= 2 {
			emotion.Synthesis(msgArr[0], msg)
		}
		return
	}
	// fmt.Println(self_id)
	repeat.CheckRepeat(msg)
}
