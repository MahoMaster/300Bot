package request

import "300Bot/send"

func CheckType(msg map[string]interface{}) {
	switch msg["request_type"] {
	case "friend":
		friend(msg)
	case "group":
		group(msg)
	default:

	}
}

// 加好友请求
func friend(msg map[string]interface{}) {
	send.SendPrivate(675559614, "有人加好友")
}

// 加群请求
func group(msg map[string]interface{}) {

}
