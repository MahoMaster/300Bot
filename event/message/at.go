package message

import (
	"300Bot/function/chatGPT"
	"300Bot/model"
	"300Bot/send"
	"strconv"
	"strings"
)

var AtList []model.At

func init() {
	updateAtList()
}

func updateAtList() {
	AtList = model.GetAtList()
}

func checkAtWords(msgStr string, msg map[string]interface{}) bool {
	for _, value := range AtList {
		if strings.Contains(msgStr, value.Keyword) {
			reply := ""
			if value.Need_at == 1 {
				msgIdStr := strconv.FormatFloat(msg["message_id"].(float64), 'f', -1, 64)
				reply = reply + "[CQ:reply,id=" + msgIdStr + "]"
			}
			if value.Need_qq == 1 {
				qqStr := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
				value.Reply = strings.Replace(value.Reply, "${qqStr}", qqStr, -1)
			}
			reply = reply + value.Reply
			send.SendGroupPost(msg["group_id"].(float64), reply)
			return true
		}
	}

	chatGPT.AddPlan(msgStr, msg)
	// if len(res) != 0 {
	// send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res))
	return true
	// }

	// return false
}
