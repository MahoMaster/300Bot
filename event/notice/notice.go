package notice

func CheckType(msg map[string]interface{}) {
	switch msg["notice_type"] {
	case "group_upload":
		groupUpload(msg)

	case "group_admin":
		groupAdmin(msg)

	case "group_decrease":
		groupDecrease(msg)

	case "group_ban":
		groupBan(msg)

	case "group_increase":
		groupIncrease(msg)

	case "friend_add":
		friendAdd(msg)

	case "group_recall":
		groupRecall(msg)

	case "friend_recall":
		friendRecall(msg)

	case "notify":
		switch msg["sub_type"] {
		case "lucky_king":
			luckyKing(msg)

		case "poke":
			poke(msg)

		case "honor":
			honor(msg)

		default:

		}

	default:

	}
}

// 群文件上传
func groupUpload(msg map[string]interface{}) {

}

// 群管理员变动
func groupAdmin(msg map[string]interface{}) {

}

// 群成员减少
func groupDecrease(msg map[string]interface{}) {

}

// 群成员增加
func groupIncrease(msg map[string]interface{}) {

}

// 群禁言
func groupBan(msg map[string]interface{}) {

}

// 好友添加
func friendAdd(msg map[string]interface{}) {

}

// 群消息撤回
func groupRecall(msg map[string]interface{}) {

}

// 好友消息撤回
func friendRecall(msg map[string]interface{}) {

}

// 群内戳一戳
func poke(msg map[string]interface{}) {

}

// 群红包运气王
func luckyKing(msg map[string]interface{}) {

}

// 群成员荣誉变更
func honor(msg map[string]interface{}) {

}
