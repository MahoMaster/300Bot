package message

import (
	"300Bot/send"
	"fmt"
)

func CheckType(msg map[string]interface{}) {
	switch msg["message_type"] {
	case "private":
		private(msg)
		break
	case "group":
		group(msg)
		break
	default:
		break
	}
}

//私聊消息
func private(msg map[string]interface{}) {
	fmt.Println("私聊消息", msg["raw_message"])
	send.Send(msg["raw_message"].(string))
}

//群消息
func group(msg map[string]interface{}) {
	fmt.Println("群消息", msg["raw_message"])
}
