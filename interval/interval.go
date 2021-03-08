package interval

import (
	"300Bot/conf"
	"300Bot/function/heros"
	"300Bot/send"
	"300Bot/store"
	"300Bot/util"
	"encoding/json"
	"log"
	"strconv"

	"github.com/robfig/cron"
)

var host = "http://127.0.0.1:" + conf.Config.ApiPort
var tomorrow int64
var c *cron.Cron

func init() {
	c = cron.New()
	c.Start()
	//定义定时器
	timeInterval()
}

func timeInterval() {
	log.Println("定时事件注册")
	// 每天七点通知天气
	// spec := "0 7 * * *"
	// c.AddFunc(spec, func() {
	// 	sendWether()
	// })
	spec1 := "30 9 * * *"
	c.AddFunc(spec1, func() {
		sayGoodMorning()
	})
	spec2 := "30 2 * * *"
	c.AddFunc(spec2, func() {
		heros.GetDailyData()
	})
	// heros.GetDailyData()
	// sendWether()
	// sendLike()
}

func sayGoodMorning() {
	for _, value := range store.GroupList {
		groupstr := strconv.FormatFloat(value.Group_id, 'f', -1, 64)
		var honor map[string]interface{}
		err := json.Unmarshal(util.HttpGet(host+"/get_group_honor_info?group_id="+groupstr+"&type=talkative"), &honor)
		if err != nil {
			log.Println(err)
			continue
		}
		if honor["retcode"].(float64) == 0 {
			if honor["data"].(map[string]interface{})["current_talkative"] != nil {
				dragonKing := honor["data"].(map[string]interface{})["current_talkative"].(map[string]interface{})
				qqStr := strconv.FormatFloat(dragonKing["user_id"].(float64), 'f', -1, 64)
				count := strconv.FormatFloat(dragonKing["day_count"].(float64), 'f', -1, 64)
				if qqStr != conf.Config.BotQQ {
					send.SendGroupPost(value.Group_id, `[CQ:at,qq=`+qqStr+`] 哦哈哟，龙王大哥哥~ 你已经蝉联龙王`+count+`天了。加油啊，龙王大哥哥，继续水群摩多摩多`)
				}
			}
		}
	}
}

func sendLike() {
	send.SendLike("675559614", 10)
}

// func sendWether() {
// 	for _, value := range store.GroupList {
// 		// SendGroup(value.Group_id, "测试7点准时定时发送消息")

// 		// 获取群成员信息，本来想推送天气，获取不到地区。gg
// 		// groupstr := strconv.FormatFloat(value.Group_id, 'f', -1, 64)
// 		// body := util.HttpGet(host + "/get_group_member_list?group_id=" + groupstr)
// 		data := make(map[string]interface{})
// 		data["group_id"] = value.Group_id

// 		var groupMenberList map[string]interface{}
// 		err := json.Unmarshal(util.HttpPost(host+"/get_group_member_list", data), &groupMenberList)
// 		if err != nil {
// 			fmt.Println(err)
// 			continue
// 		}
// 		// fmt.Println(groupMenberList)
// 		if groupMenberList["retcode"].(float64) == 0 {

// 			for _, value := range groupMenberList["data"].([]interface{}) {
// 				temp := value.(map[string]interface{})
// 				groupstr := strconv.FormatFloat(temp["group_id"].(float64), 'f', -1, 64)
// 				qqstr := strconv.FormatFloat(temp["user_id"].(float64), 'f', -1, 64)
// 				var menberInfo map[string]interface{}
// 				err = json.Unmarshal(util.HttpGet(host+"/get_group_member_info?group_id="+groupstr+"&user_id="+qqstr), &menberInfo)
// 				if err != nil {
// 					fmt.Println(err)
// 					continue
// 				}
// 				if menberInfo["retcode"].(float64) == 0 {
// 					fmt.Println(menberInfo)
// 					// info := menberInfo["data"].(map[string]interface{})
// 					// fmt.Println(info)
// 				}
// 			}
// 		}
// 	}
// }
