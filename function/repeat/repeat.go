package repeat

import (
	"300Bot/send"
	"300Bot/util"
	"path/filepath"
	"time"

	"github.com/wxnacy/wgo/arrays"
)

const repeatCD = 180

type group struct {
	//复读的人的qq
	UserIds []float64
	//复读的消息
	LastMessage string

	// 用于判断的消息(解决图片复读)
	RawMessage string
	//复读次数
	Count int
	//是否已复读
	HasRepeat bool
	//本群多久内不再复读
	CD int32
}

var (
	repeat map[float64]*group
)

func init() {
	repeat = make(map[float64]*group)
}

func CheckRepeat(msg map[string]interface{}) {
	rawMessage := ""
	if msg["raw_message"] != nil {
		rawMessage = msg["raw_message"].(string)

		if util.IsCQCode(rawMessage) {
			cqMap, err := util.ParseCQCode(rawMessage)
			if err == nil {
				if cqMap["type"] == "image" {
					// 如果是图片，只需要比较图片的file
					if _, ok := cqMap["url"]; ok {
						// 如果有url,以url为准
						rawMessage = cqMap["url"]
					} else {
						file := cqMap["file"]
						rawMessage = file
					}

				}
			}
		}
	}

	if _, ok := repeat[msg["group_id"].(float64)]; ok {
		//判断消息是否相同
		if rawMessage != repeat[msg["group_id"].(float64)].RawMessage {

			//及时修复复读机并恢复复读
			if repeat[msg["group_id"].(float64)].Count > 5 {
				// rand.Seed(time.Now().Unix())
				random := util.RandInt(1, 100)
				if random > 80 {
					repeat[msg["group_id"].(float64)].Count = 1
					// send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=0c9df9e9aaa98350bb28c1ca2661c5e0.image]`)
					name := "./static/repeat/xf1.jpg"
					path, _ := filepath.Abs(name)
					send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=file:///`+path+`]`)
					go func() {
						time.Sleep(1 * time.Second)
						send.SendGroupPost(msg["group_id"].(float64), repeat[msg["group_id"].(float64)].LastMessage)
					}()
					return
				}
			}

			repeat[msg["group_id"].(float64)].LastMessage = msg["raw_message"].(string)
			repeat[msg["group_id"].(float64)].RawMessage = rawMessage
			repeat[msg["group_id"].(float64)].Count = 1
			repeat[msg["group_id"].(float64)].HasRepeat = false
			repeat[msg["group_id"].(float64)].UserIds = []float64{msg["user_id"].(float64)}
			return
		}
		//判断复读的人是否存在
		if arrays.ContainsFloat(repeat[msg["group_id"].(float64)].UserIds, msg["user_id"].(float64)) == -1 {
			repeat[msg["group_id"].(float64)].UserIds = append(repeat[msg["group_id"].(float64)].UserIds, msg["user_id"].(float64))
		} else {
			return
		}
		//计数
		repeat[msg["group_id"].(float64)].Count = repeat[msg["group_id"].(float64)].Count + 1

		//达到次数且未复读过
		if repeat[msg["group_id"].(float64)].Count >= 2 && !repeat[msg["group_id"].(float64)].HasRepeat {
			//判断复读CD
			now := int32(time.Now().Unix())
			if now >= repeat[msg["group_id"].(float64)].CD {
				random := util.RandInt(1, 100)
				if random > 30 {
					//复读
					send.SendGroupPost(msg["group_id"].(float64), repeat[msg["group_id"].(float64)].LastMessage)
					repeat[msg["group_id"].(float64)].CD = now + repeatCD
					repeat[msg["group_id"].(float64)].HasRepeat = true
				}
			}
		}
		//及时砸了复读机并打断复读
		if repeat[msg["group_id"].(float64)].Count > 3 {
			random := util.RandInt(1, 100)
			if random > 60 {
				if random < 80 {
					name := "./static/repeat/dd1.jpg"
					path, _ := filepath.Abs(name)
					send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=file:///`+path+`]`)
					// 					send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=62653a3deddd41d3d6f117d8744a4803.image]
					// 复读姬爬`)
				} else {
					name := "./static/repeat/dd3.jpg"
					path, _ := filepath.Abs(name)
					send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=file:///`+path+`]`)
					// 					send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=39fe06f12c13d5aa0d7502df0f8a37a5.image]
					// 复读复读复读`)
				}
				repeat[msg["group_id"].(float64)].LastMessage = ""
				repeat[msg["group_id"].(float64)].RawMessage = ""
				repeat[msg["group_id"].(float64)].Count = 1
				repeat[msg["group_id"].(float64)].HasRepeat = true
				repeat[msg["group_id"].(float64)].UserIds = []float64{msg["user_id"].(float64)}
				return
			}
		}
		//及时砸了复读机并打断复读
		if repeat[msg["group_id"].(float64)].Count > 8 {
			random := util.RandInt(1, 100)
			if random > 80 {
				// send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=39ec988d2b574821cbd76a5cef2de6df.image]`)
				name := "./static/repeat/dd2.jpg"
				path, _ := filepath.Abs(name)
				send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=file:///`+path+`]
复读复读复读`)
				repeat[msg["group_id"].(float64)].LastMessage = ""
				repeat[msg["group_id"].(float64)].RawMessage = ""
				repeat[msg["group_id"].(float64)].Count = 1
				repeat[msg["group_id"].(float64)].HasRepeat = true
				repeat[msg["group_id"].(float64)].UserIds = []float64{msg["user_id"].(float64)}
			}
		}
	} else {
		repeat[msg["group_id"].(float64)] = &group{
			UserIds:     []float64{msg["user_id"].(float64)},
			LastMessage: msg["raw_message"].(string),
			RawMessage:  rawMessage,
			Count:       1,
			HasRepeat:   false,
			CD:          0,
		}
	}
}
