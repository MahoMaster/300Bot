package heros

import (
	"300Bot/model"
	"300Bot/send"
	"300Bot/util"
	"strconv"
)

func CheckIn(msg map[string]interface{}) {
	flag := model.CheckIn(msg["user_id"].(float64))
	if flag {
		send.SendGroupPost(msg["group_id"].(float64), `签到成功，积分+15`)
	} else {
		send.SendGroupPost(msg["group_id"].(float64), `今日已签到`)
	}
}

func GetUserInfo(msg map[string]interface{}) {
	user := model.GetUserInfo(msg["user_id"].(float64))
	time := "未曾签到"
	if user.Check_in != 0 {
		time = util.Time2Str(user.Check_in)
	}
	template := msg["sender"].(map[string]interface{})["nickname"].(string) + `
剩余积分:` + strconv.Itoa(user.Points) + `,
上次签到:` + time + `,
当前底图:底图` + strconv.Itoa(user.Imgbackground_set) + `,
不知道说啥了,
摩多摩多`
	send.SendGroupPost(msg["group_id"].(float64), template)
}
