package station

import (
	"300Bot/send"
	"regexp"
	"strconv"
	"strings"
	"time"
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

var lastSubmitTime = make(map[string]time.Time)

func CheckSubmitRoom(info string, msg map[string]interface{}) bool {
	info = strings.TrimSpace(info)
	if info == "" {
		return false
	}

	qqStr := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)

	// 检查信息内是否存在车牌并筛选出来
	re := regexp.MustCompile(`\b\d{5,6}\b`)
	number := re.FindString(info)
	if number == "" {
		return false
	}

	// info光车牌好像调api会报错，加个字
	if number == info {
		info = info + "来"
	}

	// 一些base64或者其他的一些杂七杂八的，偶尔会出现字符串里碰巧出现一个五六位的数字，但这些一般长度都很长，做个长度限制能杜绝大部分的
	if len(info) > 30 {
		return false
	}

	//一分钟只能发一次车
	lastTime, ok := lastSubmitTime[qqStr]
	if ok && time.Since(lastTime) < time.Minute {
		return false
	}
	lastSubmitTime[qqStr] = time.Now()

	err := SubmitRoom(number, qqStr, info)
	if err != nil {
		send.SendGroup(msg["group_id"].(float64), err.Error())
	} else {
		send.SendGroup(msg["group_id"].(float64), "已发车")
	}
	return true
}
