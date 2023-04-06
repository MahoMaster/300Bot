package chatGPT

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
	"strings"
)

var gi = goroutineNew(2)

func GetUserStory(name string, storyStr string, qq string, msg map[string]interface{}) {
	gi.goroutineRun(func() {

		//模板
		template := name + `是一名修仙者，` + storyStr + `（其中灵根数最少越好，五灵根非常平凡，单灵根万里挑一），请在70字内随机编写出` + name + `的身世故事，描写到准备踏入修仙之前即可`

		res, err := JustChatGpt(template, qq)
		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			immortalModel.LogUserStoryByQQ(qq, res.Choices[0].Message.Content)
		}
	})
}
