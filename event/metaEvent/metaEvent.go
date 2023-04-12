package metaEvent

import "log"

func CheckType(msg map[string]interface{}) {
	switch msg["meta_event_type"] {
	case "heartbeat":
		heartbeat(msg)

	case "lifecycle":
		lifecycle(msg)

	default:

	}
}

//心跳
func heartbeat(msg map[string]interface{}) {

}

//生命周期
func lifecycle(msg map[string]interface{}) {
	if msg["sub_type"] == "connect" {
		log.Println("连接成功,self_id:", msg["self_id"].(float64))
	}
}
