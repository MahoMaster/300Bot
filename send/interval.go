package send

import (
	"300Bot/store"
	"300Bot/util"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/robfig/cron"
)

var tomorrow int64
var c *cron.Cron

func init() {
	c = cron.New()
	c.Start()
	//定义定时器
	timeInterval()
}

func timeInterval() {
	// 每天七点通知天气
	// spec := "0 7 * * *"
	// c.AddFunc(spec, func() {
	// 	sendWether()
	// })
	spec := "30 9 * * *"
	c.AddFunc(spec, func() {
		sayGoodMorning()
	})

}

func sayGoodMorning() {
	for _, value := range store.GroupList {
		groupstr := strconv.FormatFloat(value.Group_id, 'f', -1, 64)
		var honor map[string]interface{}
		err := json.Unmarshal(util.HttpGet(host+"/get_group_honor_info?group_id="+groupstr+"&type=talkative"), &honor)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if honor["retcode"].(float64) == 0 {
			if honor["data"].(map[string]interface{})["current_talkative"] != nil {
				dragonKing := honor["data"].(map[string]interface{})["current_talkative"].(map[string]interface{})
				qqStr := strconv.FormatFloat(dragonKing["user_id"].(float64), 'f', -1, 64)
				count := strconv.FormatFloat(dragonKing["day_count"].(float64), 'f', -1, 64)

				SendGroupPost(value.Group_id, `[CQ:at,qq=`+qqStr+`] 哦哈哟，龙王大哥哥~ 你已经蝉联龙王`+count+`天了。加油啊，龙王大哥哥，继续水群摩多摩多`)
			}
		}
	}
}

// func sendWether() {
// 	for _, value := range store.GroupList {
// 		// SendGroup(value.Group_id, "测试7点准时定时发送消息")

// 		// 获取群成员信息，本来想推送天气，获取不到地区。gg
// 		// groupstr := strconv.FormatFloat(value.Group_id, 'f', -1, 64)
// 		// body := util.HttpGet(host + "/get_group_member_list?group_id=" + groupstr)
// 		// var groupMenberList map[string]interface{}
// 		// err := json.Unmarshal(body, &groupMenberList)
// 		// if err != nil {
// 		// 	fmt.Println(err)
// 		// 	continue
// 		// }
// 		// if groupMenberList["retcode"].(float64) == 0 {

// 		// 	for _, value := range groupMenberList["data"].([]interface{}) {
// 		// 		temp := value.(map[string]interface{})
// 		// 		groupstr := strconv.FormatFloat(temp["group_id"].(float64), 'f', -1, 64)
// 		// 		qqstr := strconv.FormatFloat(temp["user_id"].(float64), 'f', -1, 64)
// 		// 		var menberInfo map[string]interface{}
// 		// 		err = json.Unmarshal(util.HttpGet(host+"/get_group_member_info?group_id="+groupstr+"&user_id="+qqstr), &menberInfo)
// 		// 		if err != nil {
// 		// 			fmt.Println(err)
// 		// 			continue
// 		// 		}
// 		// 		if menberInfo["retcode"].(float64) == 0 {
// 		// 			info := menberInfo["data"].(map[string]interface{})
// 		// 			fmt.Println(info)
// 		// 		}
// 		// 	}
// 		// }
// 	}
// }
