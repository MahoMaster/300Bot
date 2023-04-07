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
		template := name + `是一名修仙者，背景故事关键词为:` + storyStr + `（五灵根不如四灵根，四灵根不如三灵根，三灵根不如双灵根，双灵根不如单灵根，灵根越少越好并且越稀有），请在70字内随机编写出` + name + `的身世故事，描写到准备踏入修仙之前即可.尽可能描述出关键词相关且请回避父母双亡等剧情，同时请不要出现平凡人却被人看重资质等矛盾剧情`

		res, err := JustChatGpt(template, qq)
		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			immortalModel.LogUserStoryByQQ(qq, res.Choices[0].Message.Content)
		}
	})
}
