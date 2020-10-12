package message

import (
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
		if strings.Index(msgStr, value.Keyword) != -1 {
			msgIdStr := strconv.FormatFloat(msg["message_id"].(float64), 'f', -1, 64)
			send.SendGroupPost(msg["group_id"].(float64), "[CQ:reply,id="+msgIdStr+"]"+value.Reply)
			return true
		}
	}
	return false
}
