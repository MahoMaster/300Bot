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

func SendGroupPost(group float64, msg string) {
	groupstr := strconv.FormatFloat(group, 'f', -1, 64)
	fmt.Println("发送消息到群" + groupstr + ":" + msg)

	// var data map[string]interface{}
	data := make(map[string]interface{})
	data["group_id"] = group
	data["message"] = msg

	util.HttpPost(host+"/send_group_msg", data)

}

func SendGift(group float64, qq string, num int) {
	gitf := fmt.Sprintf("[CQ:gift,qq=%s,id=%d]", qq, num)
	SendGroupPost(group, gitf)
}

func SendPoke(group float64, qq string) {
	poke := fmt.Sprintf("[CQ:poke,qq=%s]", qq)
	SendGroupPost(group, poke)
}
