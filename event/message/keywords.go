package message

import (
	"300Bot/function/img"
	"300Bot/send"
)

func checkKeywords(keyword string, msgStr string, msg map[string]interface{}) bool {
	switch keyword {
	case "help", "使用说明", "帮助":
		send.SendGroup(msg["group_id"].(float64), "http://www.mahomaster.com:3000/Maho/300Bot/src/master/doc")
		return true
	case "来张涩图", "色图", "来张色图", "涩图", "整点二次元":
		img.SendOneImg(msg)
		return true
	default:
		return false
	}
}
