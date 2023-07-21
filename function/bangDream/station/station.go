package station

import (
	"300Bot/send"
	"regexp"
	"strconv"
)

func AskForRoom(msg map[string]interface{}) {
	rooms := GetRoomList()
	if len(rooms) == 0 {
		send.SendGroupPost(msg["group_id"].(float64), "myc")
		return
	}
	send.SendGroupPost(msg["group_id"].(float64), "yc [CQ:at,qq=675559614] 完善一下")
}

func CheckSubmitRoom(info string, msg map[string]interface{}) {
	qqStr := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	re := regexp.MustCompile(`\b\d{5,6}\b`)
	number := re.FindString(info)
	if number == "" {
		return
	}

	err := SubmitRoom(number, qqStr, info)
	if err != nil {
		send.SendGroup(msg["group_id"].(float64), err.Error())
	} else {
		send.SendGroup(msg["group_id"].(float64), "已发车")
	}
}
