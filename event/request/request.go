package request

import (
	"300Bot/model"
	"300Bot/send"
	"300Bot/store"
)

func CheckType(msg map[string]interface{}) string {
	switch msg["request_type"] {
	case "friend":
		return friend(msg)

	case "group":
		group(msg)
		return ""
	default:
		return ""
	}
}

// 加好友请求
func friend(msg map[string]interface{}) string {
	send.SendPrivate(675559614, "有人加好友")

	var res = make(map[string]interface{})
	res["approve"] = true
	send.SendQuickOperation(res, msg)
	var a model.QQFriend
	a.User_id = msg["user_id"].(float64)
	store.QQFriendList = append(store.QQFriendList, a)
	return ""
}

// 加群请求
func group(msg map[string]interface{}) {

}
