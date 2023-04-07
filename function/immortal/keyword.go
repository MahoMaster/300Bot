package immortal

import (
	"300Bot/send"
	"strconv"
	"strings"
)

func CheckKeywords(msgStr string, msg map[string]interface{}) bool {
	qq := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	msgStr = msgStr[1:]
	msgArr := strings.Fields(msgStr)
	keyword := msgArr[0]
	switch keyword {
	case "帮助", "使用说明", "help":
		send.SendGroupPost(msg["group_id"].(float64), "http://www.mahomaster.com:3000/Maho/300Bot/src/master/doc/immortal.md")
		return true
	case "创建角色", "生成角色":
		flag, canDel := CheckUserByQQ(qq)
		if flag {
			if !canDel {
				send.SendGroupPost(msg["group_id"].(float64), `请勿在转世前重复创建角色`)
			} else {
				send.SendGroupPost(msg["group_id"].(float64), `请勿在转世/删除角色前重复创建角色`)
			}

			return true
		}
		if len(msgArr) < 2 {
			send.SendGroupPost(msg["group_id"].(float64), `参数错误`)
			return true
		}
		CreateUser(qq, msgArr[1], msg)
		return true
	case "删除角色":
		err := DelUserByQQ(qq)
		if err != nil {
			send.SendGroupPost(msg["group_id"].(float64), err.Error())
		} else {
			send.SendGroupPost(msg["group_id"].(float64), `删除成功`)
		}
		return true
	case "我的资料":
		err := GetUserAllInfoByQQ(qq, msg)
		if err != nil {
			send.SendGroupPost(msg["group_id"].(float64), err.Error())
			return true
		}
		return true
	case "挖矿":
		times := 1
		var err error
		if len(msgArr) > 1 {
			times, err = strconv.Atoi(msgArr[1])
			if err != nil {
				send.SendGroupPost(msg["group_id"].(float64), "参数错误")
				return true
			}
		}
		err = Mining(qq, msg, times)
		if err != nil {
			send.SendGroupPost(msg["group_id"].(float64), err.Error())
			return true
		}
		return true
	case "系统功法商城":
		name := ""
		var err error
		if len(msgArr) >= 3 {
			name = msgArr[2]
		}
		page := 1
		if len(msgArr) >= 2 {
			page, err = strconv.Atoi(msgArr[1])
			if err != nil {
				send.SendGroupPost(msg["group_id"].(float64), "参数错误")
				return true
			}
		}
		err = GetSkillAdminShop(qq, msg, name, page)
		if err != nil {
			send.SendGroupPost(msg["group_id"].(float64), err.Error())
			return true
		}
		return true
	case "查询功法":
		if len(msgArr) < 2 {

			send.SendGroupPost(msg["group_id"].(float64), "参数错误")
			return true
		}
		err := GetSkillDetail(msgArr[1], msg)
		if err != nil {
			send.SendGroupPost(msg["group_id"].(float64), err.Error())
			return true
		}
		return true
	case "购买功法":
		if len(msgArr) < 2 {

			send.SendGroupPost(msg["group_id"].(float64), "参数错误")
			return true
		}
		err := BuySkill(qq, msgArr[1], msg)
		if err != nil {
			send.SendGroupPost(msg["group_id"].(float64), err.Error())
			return true
		}
		return true
	case "test":
		send.SendGroupPost(msg["group_id"].(float64), "test")
		return true
	default:

		return false
	}
}
