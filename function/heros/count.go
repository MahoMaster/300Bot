package heros

import (
	"300Bot/model/herosModel"
	"300Bot/send"
	"300Bot/util"
	"fmt"
	"strconv"
	"time"
)

func GetBattleCount(keyword string, descType string, limit string, msg map[string]interface{}) {
	t := time.Now()
	timeInt := int64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()) - 60*60*24 //凌晨时间
	orderType := 0                                                                                        //0为出场排行，1为胜率排行
	switch keyword {
	case "单日出场排行":
		break
	case "单日胜率排行":
		orderType = 1
		break
	case "三日出场排行":
		timeInt = timeInt - 60*60*24*2
		break
	case "三日胜率排行":
		timeInt = timeInt - 60*60*24*2
		orderType = 1
		break
	case "一周出场排行":
		timeInt = timeInt - 60*60*24*6
		break
	case "一周胜率排行":
		timeInt = timeInt - 60*60*24*6
		orderType = 1
		break
	default:
		break
	}
	timeStr := util.Time2Str(timeInt)
	countList := herosModel.GetHerosWin(timeStr, orderType, descType, limit)
	template := ``
	for index, value := range countList {
		if value.Name == "" {
			value.Name = "鬼知道什么偷跑英雄"
		}
		template += `第` + strconv.Itoa(index+1) + `名
[CQ:image,file=` + value.Icon + `] ` + value.Name + ` ,胜率: ` + strconv.Itoa(value.Win) + `/` + strconv.Itoa(value.Win+value.Lose) + ` ` + fmt.Sprintf("%.2f", value.Rate) + `%

`
	}
	send.SendGroupPost(msg["group_id"].(float64), template)
}

// func GetBattleCount(keyword string, descType string, limit string, msg map[string]interface{}) {
// 	// switch keyword {
// 	// case "单日出场排行":
// 	// 	break
// 	// case "单日胜率排行":
// 	// 	orderType = 1
// 	// 	break
// 	// case "三日出场排行":
// 	// 	timeInt = timeInt - 60*60*24*2
// 	// 	break
// 	// case "三日胜率排行":
// 	// 	timeInt = timeInt - 60*60*24*2
// 	// 	orderType = 1
// 	// 	break
// 	// case "一周出场排行":
// 	// 	timeInt = timeInt - 60*60*24*6
// 	// 	break
// 	// case "一周胜率排行":
// 	// 	timeInt = timeInt - 60*60*24*6
// 	// 	orderType = 1
// 	// 	break
// 	// default:
// 	// 	break
// 	// }
// 	timeStr := util.Time2Str(timeInt)
// 	countList := herosModel.GetHerosWin(timeStr, orderType, descType, limit)
// 	template := ``
// 	for index, value := range countList {
// 		if value.Name == "" {
// 			value.Name = "鬼知道什么偷跑英雄"
// 		}
// 		template += `第` + strconv.Itoa(index+1) + `名
// [CQ:image,file=` + value.Icon + `] ` + value.Name + ` ,胜率: ` + strconv.Itoa(value.Win) + `/` + strconv.Itoa(value.Win+value.Lose) + ` ` + fmt.Sprintf("%.2f", value.Rate) + `%

// `
// 	}
// 	send.SendGroupPost(msg["group_id"].(float64), template)
// }
