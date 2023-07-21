package message

import (
	"300Bot/conf"
	"300Bot/function/ban"
	"300Bot/function/bangDream/station"
	"300Bot/function/chatGPT"
	"300Bot/function/emotion"
	"300Bot/function/heros"
	"300Bot/function/img"
	"300Bot/function/music"
	"300Bot/function/wether"
	"300Bot/send"
	"300Bot/store"
	"path/filepath"
	"strconv"
	"strings"
)

func checkKeywords(keyword string, msgStr string, msg map[string]interface{}) bool {
	switch keyword {
	case "help", "使用说明", "帮助":
		send.SendGroupPost(msg["group_id"].(float64), "http://gogs.yugi.cc/Maho/300Bot/src/master/doc")
		return true
	case "来张涩图", "色图", "来张色图", "涩图", "整点二次元":
		img.SendOneImg(msg)
		return true
	case "不够色":
		send.SendGroupPost(msg["group_id"].(float64), "钱都不给还想看好康的？[CQ:face,id=176]")
		return true
	case "点歌":
		msgArr := strings.Split(msgStr, keyword)
		// fmt.Println(len(msgArr))
		if len(msgArr) < 2 {
			send.SendGroupPost(msg["group_id"].(float64), "参数错误")
		} else if msgArr[1] == "" || msgArr[1] == " " {
			send.SendGroupPost(msg["group_id"].(float64), "请输入关键词")
		} else {
			music.ShareMusic(msgArr[1], msg)
		}
		return true
	case "天气", "查天气", "当前天气", "天气预报":
		msgArr := strings.Split(msgStr, keyword)
		if msgArr[1] == "" || msgArr[1] == " " {
			send.SendGroupPost(msg["group_id"].(float64), "请输入关键词")
		} else {
			wether.GetCityWether(msgArr[1], msg)
		}
		return true
	case "设置底图", "底图设置":
		msgArr := strings.Split(msgStr, keyword)
		if msgArr[1] == "" || msgArr[1] == " " {
			send.SendGroupPost(msg["group_id"].(float64), "参数错误")
		} else {
			emotion.SetImgBackground(msgArr[1], msg)
		}
		return true
	case "底图目录":
		path, _ := filepath.Abs("./static/imgBackground/0.jpg")
		send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=file:///`+path+`]`)
		return true
	case "免费礼物", "送礼":
		send.SendGroupPost(msg["group_id"].(float64), "暂时失效")
		return true
		// msgArr := strings.Split(msgStr, keyword)
		// if msgArr[1] == "" || msgArr[1] == " " {
		// 	send.SendGroupPost(msg["group_id"].(float64), "参数错误")
		// } else {
		// 	present.SendGift(msgArr[1], msg)
		// }
		// return true
	case "签到", "打卡":
		heros.CheckIn(msg)
		return true
	case "积分查询":
		heros.GetUserInfo(msg)
		return true
	case "生成图片":
		msgArr := strings.Split(msgStr, keyword)
		if msgArr[1] == "" || msgArr[1] == " " {
			send.SendGroupPost(msg["group_id"].(float64), "请输入关键词")
		} else {
			chatGPT.AddImgPlan(msgArr[1], msg)
		}
		return true
	case "bot测试":
		// station.AskForRoom(msg)
		// defer func() {
		// 	if info := recover(); info != nil {
		// 		fmt.Println("触发了宕机", info)
		// 	} else {
		// 		fmt.Println("芜锁胃")
		// 	}
		// }()
		// send.SendPoke(msg["group_id"].(float64), "&#91;骰子&#93;")
		// send.SendGroupPost(msg["group_id"].(float64), "[CQ:dice]")
		// res := chatGPT.AskForChatGPT(msgStr, msg["user_id"].(float64))
		// if len(res) != 0 {
		// 	send.SendGroupPost(msg["group_id"].(float64), res)
		// }
		// panic("tt")
		// chatGPT.AddImgPlan("一个二次元女人", msg)
		return true
	case "Ban", "禁用", "ban", "解除禁用":
		flag := checkManage(msg)
		if !flag {
			send.SendGroupPost(msg["group_id"].(float64), "你有个锤锤权限，爬")
			return true
		}
		msgArr := strings.Split(msgStr, keyword)
		if msgArr[1] == "" || msgArr[1] == " " {
			send.SendGroupPost(msg["group_id"].(float64), "参数错误")
		} else {
			if keyword == "解除禁用" {
				ban.BanSomeOne(msgArr[1], 0, msg)
			} else {
				ban.BanSomeOne(msgArr[1], 1, msg)
			}

		}
		return true
	case "设置人格":
		msgArr := strings.Split(msgStr, keyword)
		if msgArr[1] == "" || msgArr[1] == " " {
			send.SendGroupPost(msg["group_id"].(float64), "参数错误")
		} else {
			chatGPT.SetPersonality(msgArr[1], msg)
		}
		return true

	case "ycm", "有车吗":
		station.AskForRoom(msg)
		return true
	case "发车":
		msgArr := strings.Split(msgStr, keyword)
		if msgArr[1] == "" || msgArr[1] == " " {
			send.SendGroupPost(msg["group_id"].(float64), "参数错误")
		} else {
			station.CheckSubmitRoom(msgArr[1], msg)
		}
		return true
	// case "单日出场排行", "单日胜率排行", "三日出场排行", "三日胜率排行", "一周出场排行", "一周胜率排行":
	// 	msgArr := strings.Split(msgStr, keyword)
	// 	param := strings.TrimSpace(msgArr[1])
	// 	paramArr := strings.Split(param, " ")
	// 	limit := "20"
	// 	descType := "desc"

	// 	if len(paramArr) >= 1 {
	// 		if paramArr[0] != "" && paramArr[0] != " " {
	// 			limit = strings.TrimSpace(paramArr[0])
	// 			limitInt, err := strconv.Atoi(limit)
	// 			if err != nil {
	// 				send.SendGroupPost(msg["group_id"].(float64), "参数错误")
	// 				return true
	// 			}
	// 			if limitInt <= 0 {
	// 				send.SendGroupPost(msg["group_id"].(float64), "参数错误")
	// 				return true
	// 			}
	// 			if limitInt > 20 {
	// 				send.SendGroupPost(msg["group_id"].(float64), "数据有点多了哦，以后会优化")
	// 				return true
	// 			}
	// 		}
	// 	}
	// 	if len(paramArr) >= 2 {
	// 		if paramArr[1] != "" && paramArr[1] != " " {
	// 			desc := strings.TrimSpace(paramArr[1])
	// 			descInt, err := strconv.Atoi(desc)
	// 			if err != nil {
	// 				send.SendGroupPost(msg["group_id"].(float64), "参数错误")
	// 				return true
	// 			}
	// 			if descInt == 1 {
	// 				descType = "asc"
	// 			}
	// 		}
	// 	}
	// 	// return true
	// 	heros.GetBattleCount(keyword, descType, limit, msg)
	// 	return true
	default:
		return false
	}
}

func checkManage(msg map[string]interface{}) bool {
	groupIndex := -1
	for key, value := range store.GroupList {
		if value.Group_id == msg["group_id"].(float64) {
			groupIndex = key
		}
	}
	if groupIndex == -1 {
		return false
	}
	qqstr := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)

	if qqstr == conf.Config.Manager || qqstr == strconv.FormatFloat(store.GroupList[groupIndex].Manager, 'f', -1, 64) {
		return true
	} else {
		return false
	}
}
