package message

import (
	"300Bot/model"
	"300Bot/send"
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
			send.SendGroup(msg["group_id"].(float64), value.Reply)
			return true
		}
	}
	return false
}
