package station

import (
	"300Bot/send"
	"regexp"
	"strconv"
	"strings"
)

func AskForRoom(msg map[string]interface{}) {
	rooms := GetRoomList()
	if len(rooms) == 0 {
		send.SendGroupPost(msg["group_id"].(float64), "myc")
		return
	}
	template := `------------------------
`
	for _, room := range rooms {
		template += `车牌：` + room.Number + `
介绍:` + room.Raw_message + `
------------------------`
	}
	send.SendGroupPost(msg["group_id"].(float64), template)
}

func CheckSubmitRoom(info string, msg map[string]interface{}) {
	info = strings.TrimSpace(info)
	if info == "" {
		return
	}
	qqStr := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	re := regexp.MustCompile(`\b\d{5,6}\b`)
	number := re.FindString(info)
	if number == "" {
		return
	}
	if number == info {
		send.SendGroup(msg["group_id"].(float64), "写一下车牌描述")
		return
	}
	err := SubmitRoom(number, qqStr, info)
	if err != nil {
		send.SendGroup(msg["group_id"].(float64), err.Error())
	} else {
		send.SendGroup(msg["group_id"].(float64), "已发车")
	}
}
