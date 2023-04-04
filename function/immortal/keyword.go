package immortal

import (
	"300Bot/send"
	"strconv"
	"strings"
)

func CheckKeywords(msgStr string, msg map[string]interface{}) bool {
	qq := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	msgStr = msgStr[1:]
	msgArr := strings.Fields(msgStr)
	keyword := msgArr[0]
	switch keyword {
	case "创建角色", "生成角色":
		flag := CheckUserByQQ(qq)
		if flag {
			send.SendGroupPost(msg["group_id"].(float64), `请勿在转世前重复创建角色`)
			return true
		}
		CreateUser(qq, msgArr[1], msg)
		return true
	default:

		return false
	}
}
