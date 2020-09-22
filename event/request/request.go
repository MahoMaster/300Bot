package request

func CheckType(msg map[string]interface{}) {
	switch msg["request_type"] {
	case "friend":
		friend(msg)
		break
	case "group":
		group(msg)
		break
	default:
		break
	}
}

//加好友请求
func friend(msg map[string]interface{}) {

}

//加群请求
func group(msg map[string]interface{}) {

}
