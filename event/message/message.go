package message

import (
	"300Bot/function/repeat"
	"300Bot/model"
	"300Bot/send"
	"300Bot/util"
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

var GroupList []model.Group

// var groupIdList []float64

func init() {
	updateGroupList()
}
func updateGroupList() {
	GroupList = model.GetGroupList()
	// groupIdList = make([]float64, 0)
	// for _, value := range GroupList {
	// 	groupIdList = append(groupIdList, value.Group_id)
	// }
}

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
	for key, value := range GroupList {
		if value.Group_id == msg["group_id"].(float64) {
			groupIndex = key
		}
	}
	if groupIndex == -1 {
		return
	}
	self_id := msg["self_id"].(float64)

	msgStr := msg["raw_message"].(string)

	//获取关键字

	//查询at
	self_id_str := strconv.FormatFloat(self_id, 'f', -1, 64)
	if strings.Index(msg["raw_message"].(string), "[CQ:at,qq="+self_id_str+"]") != -1 {
		msgStr = exstrings.Replace(msgStr, "[CQ:at,qq="+self_id_str+"]", "", -1)
		msgStr = util.DeletePreAndSufSpace(msgStr)
		//如果是at的关键词就直接结束
		if checkAtWords(msgStr, msg) {
			return
		}
	}
	// fmt.Println(self_id)
	repeat.CheckRepeat(msg)
}
