package interval

import (
	"300Bot/conf"
	"300Bot/send"
	"300Bot/store"
	"300Bot/util"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

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
	spec1 := "0 38 9 * * *"
	c.AddFunc(spec1, func() {
		sayGoodMorning()

		// youzanSign()
		// sendLike()
		omelet()
	})

	// youzanSign()
	// spec2 := "30 2 * * *"
	// c.AddFunc(spec2, func() {
	// 	heros.GetDailyData()
	// })
	// heros.GetDailyData()
	// sendWether()
	// sendLike()
	// sayGoodMorning()
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
		res, _ := json.Marshal(honor)
		log.Println(string(res))
		if honor["retcode"].(float64) == 0 {
			if honor["data"].(map[string]interface{})["current_talkative"] != nil {
				dragonKing := honor["data"].(map[string]interface{})["current_talkative"].(map[string]interface{})
				if dragonKing["user_id"] == nil {
					continue
				}
				qqStr := strconv.FormatFloat(dragonKing["user_id"].(float64), 'f', -1, 64)
				// count := strconv.FormatFloat(dragonKing["day_count"].(float64), 'f', -1, 64)
				description := dragonKing["description"].(string)
				if qqStr != conf.Config.BotQQ {
					send.SendGroupPost(value.Group_id, `[CQ:at,qq=`+qqStr+`] 哦哈哟，龙王大哥哥~ 你已经当龙王`+description+`。加油啊，龙王大哥哥，继续水群摩多摩多`)
				}
			}
		}
	}
}

func omelet() {

	for _, value := range store.GroupList {
		name := "./static/interval/hbd.gif"
		path, _ := filepath.Abs(name)
		send.SendGroupPost(value.Group_id, `[CQ:image,file=file:///`+path+`]`)

	}
}

func sendLike() {
	send.SendLike("675559614", 10)
}

func youzanSign() {

	defer func() {
		if info := recover(); info != nil {
			log.Println("芜锁胃", info)
		}
	}()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// 超时时间：60秒
	client := &http.Client{Timeout: 60 * time.Second, Transport: transport}
	access_token := "6e7bd80d1fb3e3c545fcfe825bc2ac"
	extraData := `{"is_weapp":1,"sid":"YZ1136597133200211968YZDwCaoUGN","version":"3.99.7.101","client":"weapp","bizEnv":"retail","uuid":"qWRjuRXwjs4iS0y1679028044231","ftime":1679028044226}`
	url := "https://h5.youzan.com/wscump/checkin/checkinV2.json?checkinId=2986433&app_id=wxce54ee7f76ebd245&kdt_id=116110226&access_token=" + access_token
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// panic(err)
		log.Println(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Extra-Data", extraData)
	req.Header.Set("Host", "h5.youzan.com")
	req.Header.Set("referer", "https://servicewechat.com/wxce54ee7f76ebd245/31/page-frame.html")
	req.Header.Set("xweb_xhr", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF XWEB/6939")
	req.Header.Set("page-path", "pages/home/dashboard/index")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	log.Println(string(result))
	var res map[string]interface{}
	err = json.Unmarshal(result, &res)
	if err != nil {
		log.Println(err)
	}
	if res["code"].(float64) == 0 {
		send.SendPrivate(675559614, "签到成功")
	} else {
		send.SendPrivate(675559614, "签到失败")
	}

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
