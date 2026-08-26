package send

import "300Bot/util"

// GetGroupMsgHistory 拉取群历史消息（message_seq=0 表示从最新开始）。
// 注意：NapCat 历史拉取依赖 QQ 客户端本地缓存，过老消息可能拉不到。
func GetGroupMsgHistory(groupId float64, count int) []byte {
	data := map[string]interface{}{
		"group_id":      groupId,
		"message_seq":   0,
		"count":         count,
		"reverse_order": false,
	}
	return util.HttpPost(host+"/get_group_msg_history", data)
}

// GetFriendMsgHistory 拉取私聊历史消息（NapCat 扩展 API）
func GetFriendMsgHistory(userId float64, count int) []byte {
	data := map[string]interface{}{
		"user_id": userId,
		"count":   count,
	}
	return util.HttpPost(host+"/get_friend_msg_history", data)
}

// GetGroupMemberInfo 获取群成员信息，用于 QQ 号 ↔ 当前昵称解析
func GetGroupMemberInfo(groupId float64, userId float64) []byte {
	data := map[string]interface{}{
		"group_id": groupId,
		"user_id":  userId,
	}
	return util.HttpPost(host+"/get_group_member_info", data)
}

// GetMsg 按 message_id 获取单条消息，用于引用/回复场景补全
func GetMsg(messageId float64) []byte {
	data := map[string]interface{}{
		"message_id": messageId,
	}
	return util.HttpPost(host+"/get_msg", data)
}
