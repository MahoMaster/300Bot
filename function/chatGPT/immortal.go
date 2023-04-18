package chatGPT

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
	"log"
	"strings"
)

// var gi = goroutineNew(1)

func GetUserStory(name string, storyStr string, qq string, msg map[string]interface{}) {
	g.goroutineRun(func() {

		//模板
		template := name + `是一名修仙者，背景故事关键词为:` + storyStr + `（五灵根不如四灵根，四灵根不如三灵根，三灵根不如双灵根，双灵根不如单灵根，灵根越少越好并且越稀有），请在70字内随机编写出` + name + `的身世故事，描写到准备踏入修仙之前即可.尽可能描述出关键词相关且请回避父母双亡等剧情，同时请不要出现平凡人却被人看重资质等矛盾剧情`

		res, err := JustChatGpt(template, qq)
		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			immortalModel.LogUserStoryByQQ(qq, res.Choices[0].Message.Content)
		}
	})
}

func LevelUpBeforeStory(name string, level1Name string, level2Name string, qq string, msg map[string]interface{}) {
	g.goroutineRun(func() {

		//模板
		template := name + `是一名修仙者，今天试图从` + level1Name + `突破到` + level2Name + `，修为从低到高分为凡人、炼气、筑基、金丹、元婴、化神，每个修为分为九层，请用30字以内随机描述一段准备突破前的心情和当时的环境描写`

		res, err := JustChatGpt(template, qq)
		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			immortalModel.LogUserStoryByQQ(qq, res.Choices[0].Message.Content)
		}
	})
}

func LevelUpIngStory(name string, level1Name string, level2Name string, qq string, msg map[string]interface{}) {
	g.goroutineRun(func() {

		//模板
		template := name + `是一名修仙者，今天试图从` + level1Name + `突破到` + level2Name + `，修为从低到高分为凡人、炼气、筑基、金丹、元婴、化神，每个修为分为九层，请用30字以内随机描述一段突破当中时的状态描写和环境描写`

		res, err := JustChatGpt(template, qq)
		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			immortalModel.LogUserStoryByQQ(qq, res.Choices[0].Message.Content)
		}
	})
}

func LevelUpResultStory(name string, level1Name string, level2Name string, qq string, succ int, get string, msg map[string]interface{}) {
	g.goroutineRun(func() {
		result := "失败了，状态跌落，需要重头再来"
		if succ == 1 {
			result = "成功了"
		}
		//模板
		template := name + `是一名修仙者，今天试图从` + level1Name + `突破到` + level2Name + `，` + result + `,` + get + `修为从低到高分为凡人、炼气、筑基、金丹、元婴、化神，每个修为分为九层，请用60字以内的一段话随机描述突破后的环境描写和心情描写`
		log.Println(template)
		res, err := JustChatGpt(template, qq)
		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			immortalModel.LogUserStoryByQQ(qq, res.Choices[0].Message.Content)
		}
	})
}
