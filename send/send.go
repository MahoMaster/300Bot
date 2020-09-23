package send

import (
	"300Bot/conf"
	"300Bot/util"
	"fmt"
	"strconv"
)

var host = "http://127.0.0.1:" + conf.Config.ApiPort

func SendPrivate(qq float64, msg string) {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	util.HttpGet(host + "/send_private_msg?user_id=" + qqstr + "&message=" + msg)
}

func SendGroup(group float64, msg string) {
	groupstr := strconv.FormatFloat(group, 'f', -1, 64)
	fmt.Println("发送消息到群" + groupstr + ":" + msg)
	util.HttpGet(host + "/send_group_msg?group_id=" + groupstr + "&message=" + msg)
}
