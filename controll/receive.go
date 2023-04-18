package controll

import (
	"300Bot/event/message"
	"300Bot/event/metaEvent"
	"300Bot/event/notice"
	"300Bot/event/request"
)

func CheckWsMsg(msg map[string]interface{}) {
	// fmt.Println(msg)
	switch msg["post_type"] {
	case "message":
		//消息事件
		message.CheckType(msg)
	case "notice":
		//通知事件
		notice.CheckType(msg)
	case "request":
		//请求事件
		request.CheckType(msg)
	case "meta_event":
		//元事件
		metaEvent.CheckType(msg)
	default:
	}
}
