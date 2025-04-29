package music

import (
	"300Bot/send"
	"300Bot/util"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

func ShareMusic(name string, msg map[string]interface{}) {
	var resp map[string]interface{}
	body := util.HttpGet("https://music.163.com/api/search/get/web?csrf_token=hlpretag=&hlposttag=&s=" + url.QueryEscape(name) + "&type=1&offset=0&total=true&limit=1")
	err := json.Unmarshal(body, &resp)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp["result"].(map[string]interface{})["songs"] != nil {
		songId := resp["result"].(map[string]interface{})["songs"].([]interface{})[0].(map[string]interface{})["id"].(float64)
		send.SendGroupPost(msg["group_id"].(float64), "[CQ:music,type=163,id="+strconv.FormatFloat(songId, 'f', -1, 64)+"]")
	} else {
		send.SendGroupPost(msg["group_id"].(float64), "未查到该歌曲")
	}
}
