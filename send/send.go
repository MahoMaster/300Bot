package send

import (
	"300Bot/conf"
	"300Bot/util"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var host = "http://" + conf.Config.ApiUrl + ":" + conf.Config.ApiPort

func SendPrivate(qq float64, msg string) {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	log.Println("私聊消息" + qqstr + ":" + msg)
	util.HttpGet(host + "/send_private_msg?user_id=" + qqstr + "&message=" + msg)
}

func SendGroup(group float64, msg string) {
	groupstr := strconv.FormatFloat(group, 'f', -1, 64)
	log.Println("发送消息到群" + groupstr + ":" + msg)
	util.HttpGet(host + "/send_group_msg?group_id=" + groupstr + "&message=" + msg)

}
func SendPrivatePost(qq float64, msg string) {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	log.Println("私聊消息" + qqstr + ":" + msg)
	payload := interface{}(msg)
	if segments, ok := msgToNapCatSegments(msg); ok {
		payload = segments
	}
	data := make(map[string]interface{})
	data["user_id"] = qq
	data["message"] = payload
	util.HttpPost(host+"/send_private_msg", data)

	if segments, ok := payload.([]map[string]interface{}); ok {
		scheduleRemoveTempPaths(segments)
	}
}

func SendPrivateImageFile(qq float64, absFilePath string) {
	log.Println("私聊图片" + strconv.FormatFloat(qq, 'f', -1, 64) + ":" + absFilePath)
	data := make(map[string]interface{})
	data["user_id"] = qq
	data["message"] = []map[string]interface{}{
		{
			"type": "image",
			"data": map[string]interface{}{
				"file": absFilePath,
			},
		},
	}
	util.HttpPost(host+"/send_private_msg", data)
	scheduleRemoveTempPath(absFilePath)
}
func SendGroupPost(group float64, msg string) {
	groupstr := strconv.FormatFloat(group, 'f', -1, 64)
	log.Println("发送消息到群" + groupstr + ":" + msg)

	// var data map[string]interface{}
	payload := interface{}(msg)
	if segments, ok := msgToNapCatSegments(msg); ok {
		payload = segments
	}
	data := make(map[string]interface{})
	data["group_id"] = group
	data["message"] = payload

	util.HttpPost(host+"/send_group_msg", data)
	if segments, ok := payload.([]map[string]interface{}); ok {
		scheduleRemoveTempPaths(segments)
	}

}

func SendPrivatePostHasGroup(qq float64, group_id float64, msg string) {
	groupstr := strconv.FormatFloat(group_id, 'f', -1, 64)
	qqStr := strconv.FormatFloat(qq, 'f', -1, 64)
	log.Println("通过群聊" + groupstr + "发送临时会话消息到" + qqStr + ":" + msg)

	// var data map[string]interface{}
	payload := interface{}(msg)
	if segments, ok := msgToNapCatSegments(msg); ok {
		payload = segments
	}
	data := make(map[string]interface{})
	data["user_id"] = qqStr
	data["group_id"] = group_id
	data["message"] = payload

	util.HttpPost(host+"/send_private_msg", data)
	if segments, ok := payload.([]map[string]interface{}); ok {
		scheduleRemoveTempPaths(segments)
	}
}
func SendGroupPostHasRes(group float64, msg string) []byte {
	groupstr := strconv.FormatFloat(group, 'f', -1, 64)
	log.Println("发送消息到群" + groupstr + ":" + msg)

	// var data map[string]interface{}
	payload := interface{}(msg)
	if segments, ok := msgToNapCatSegments(msg); ok {
		payload = segments
	}
	data := make(map[string]interface{})
	data["group_id"] = group
	data["message"] = payload

	res := util.HttpPost(host+"/send_group_msg", data)
	if segments, ok := payload.([]map[string]interface{}); ok {
		scheduleRemoveTempPaths(segments)
	}
	return res
}

func scheduleRemoveTempPaths(segments []map[string]interface{}) {
	for _, seg := range segments {
		data, ok := seg["data"].(map[string]interface{})
		if !ok {
			continue
		}
		p, ok := data["path"].(string)
		if !ok || strings.TrimSpace(p) == "" {
			continue
		}
		scheduleRemoveTempPath(p)
	}
}

func scheduleRemoveTempPath(p string) {
	tempDirAbs, err := filepath.Abs("./static/temp")
	if err != nil {
		return
	}

	absPath := p
	if !filepath.IsAbs(absPath) {
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return
		}
	}

	absPath = filepath.Clean(absPath)
	tempDirAbs = filepath.Clean(tempDirAbs)

	prefix := tempDirAbs + string(filepath.Separator)
	if !strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(prefix)) {
		return
	}

	go func(path string) {
		time.Sleep(5 * time.Second)
		os.Remove(path)
	}(absPath)
}

func msgToNapCatSegments(msg string) ([]map[string]interface{}, bool) {
	if msg == "" {
		return nil, false
	}
	if !strings.Contains(msg, "[CQ:") {
		return nil, false
	}

	var segments []map[string]interface{}
	hasImage := false

	for len(msg) > 0 {
		start := strings.Index(msg, "[CQ:")
		if start == -1 {
			if msg != "" {
				segments = append(segments, map[string]interface{}{
					"type": "text",
					"data": map[string]interface{}{"text": msg},
				})
			}
			break
		}
		if start > 0 {
			segments = append(segments, map[string]interface{}{
				"type": "text",
				"data": map[string]interface{}{"text": msg[:start]},
			})
			msg = msg[start:]
		}

		end := strings.Index(msg, "]")
		if end == -1 {
			segments = append(segments, map[string]interface{}{
				"type": "text",
				"data": map[string]interface{}{"text": msg},
			})
			break
		}

		cq := msg[:end+1]
		msg = msg[end+1:]

		parsed, err := util.ParseCQCode(cq)
		if err != nil {
			segments = append(segments, map[string]interface{}{
				"type": "text",
				"data": map[string]interface{}{"text": cq},
			})
			continue
		}
		if parsed["type"] != "image" {
			segments = append(segments, map[string]interface{}{
				"type": "text",
				"data": map[string]interface{}{"text": cq},
			})
			continue
		}

		hasImage = true
		imgData := map[string]interface{}{}
		imgData["sub_type"] = 0

		if u := strings.TrimSpace(parsed["url"]); u != "" {
			imgData["url"] = u
		}

		f := strings.TrimSpace(parsed["file"])
		if f != "" {
			if strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") {
				imgData["url"] = f
			} else if strings.HasPrefix(strings.ToLower(f), "file:///") || strings.HasPrefix(strings.ToLower(f), "file://") {
				p := f
				p = strings.TrimPrefix(p, "file:///")
				p = strings.TrimPrefix(p, "file://")
				p = strings.TrimLeft(p, "/")
				p = filepath.FromSlash(p)
				imgData["file"] = p
			} else if filepath.IsAbs(f) {
				imgData["file"] = f
			} else {
				imgData["file"] = f
			}
		}
		log.Println(imgData)
		segments = append(segments, map[string]interface{}{
			"type": "image",
			"data": imgData,
		})
	}

	if !hasImage {
		return nil, false
	}
	return segments, true
}
func SendTTS(group float64, msg string) {
	tts := fmt.Sprintf("[CQ:tts,text=%s]", msg)
	SendGroupPost(group, tts)
}

func SendGift(group float64, qq string, num int) {
	gitf := fmt.Sprintf("[CQ:gift,qq=%s,id=%d]", qq, num)
	SendGroupPost(group, gitf)
}

func SendPoke(group float64, qq string) {
	poke := fmt.Sprintf("[CQ:poke,qq=%s]", qq)
	SendGroupPost(group, poke)
}

func SendLike(qq string, times int) {
	timesStr := strconv.Itoa(times)
	fmt.Println("/send_like?user_id=" + qq + "&times=" + timesStr)
	res := util.HttpGet(host + "/send_like?user_id=" + qq + "&times=" + timesStr)
	fmt.Println(string(res))
}

func SetStarMessage(message_id float64) {
	data := make(map[string]interface{})
	data["message_id"] = message_id

	util.HttpPost(host+"/set_essence_msg", data)
	// log.Println(string(res))
}

func SendQuickOperation(data interface{}, msg map[string]interface{}) {
	res := make(map[string]interface{})
	res["context"] = msg
	res["operation"] = data

	util.HttpPost(host+"/.handle_quick_operation", res)
	// log.Println(string(res))
}

func GetQQFriendList() []byte {

	res := util.HttpPost(host+"/get_friend_list", nil)
	return res
}
