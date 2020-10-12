package message

import (
	"300Bot/function/emotion"
	"300Bot/function/img"
	"300Bot/function/music"
	"300Bot/function/wether"
	"300Bot/send"
	"path/filepath"
	"strings"
)

func checkKeywords(keyword string, msgStr string, msg map[string]interface{}) bool {
	switch keyword {
	case "help", "使用说明", "帮助":
		send.SendGroupPost(msg["group_id"].(float64), "http://www.mahomaster.com:3000/Maho/300Bot/src/master/doc")
		return true
	case "来张涩图", "色图", "来张色图", "涩图", "整点二次元":
		img.SendOneImg(msg)
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
	default:
		return false
	}
}
