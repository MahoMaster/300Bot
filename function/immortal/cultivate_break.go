package immortal

import (
	"300Bot/function/chatGPT"
	"300Bot/model/immortalModel"
	"300Bot/send"
	"300Bot/util"
	"math"
	"time"
)

func level1Up(u immortalModel.User, uc immortalModel.User_cultivate, level immortalModel.Level, msg map[string]interface{}) error {

	random := util.RandInt(1, 100)

	randomUp := float64(random * (1 + (u.Lucky-10)/10 + (u.Insight-10)/10))

	next_level, err := immortalModel.GetLevel(level.Next_level)
	if err != nil {
		return err
	}
	send.SendGroupPost(msg["group_id"].(float64), u.Name+`将开始突破`)
	go func() {
		// chatGPT.LevelUpBeforeStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2)
		// chatGPT.LevelUpIngStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2)
		if randomUp > 65 {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功，踏进了真正的修仙之路`)
			uc.Level = next_level.Id
			immortalModel.UpdateUserLevel(u.Id, uc.Level)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, msg)
		} else {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破失败,修为跌落`)
			randomLast := util.RandInt(0, uc.Aura)
			// log.Println(randomLast)
			randomLast = int(math.Floor(float64(randomLast * (1 + (u.Lucky-10)/10))))
			// log.Println(randomLast)
			if randomLast > level.Up_need_aura {
				randomLast = level.Up_need_aura
			}
			// log.Println(randomLast)
			// uc.Aura = randomLast
			immortalModel.UpdateUserAura(u.Id, randomLast-uc.Aura)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, msg)
		}

	}()

	return nil
}

func level2Up(u immortalModel.User, uc immortalModel.User_cultivate, level immortalModel.Level, msg map[string]interface{}) error {

	random := util.RandInt(1, 100)

	randomUp := float64(random * (1 + (u.Lucky-10)/10 + (u.Insight-10)/10))

	next_level, err := immortalModel.GetLevel(level.Next_level)
	if err != nil {
		return err
	}
	send.SendGroupPost(msg["group_id"].(float64), u.Name+`将开始突破到`+next_level.Name)
	go func() {
		// chatGPT.LevelUpBeforeStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2)
		// chatGPT.LevelUpIngStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2)
		if randomUp > 95 {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功`)
			uc.Level = next_level.Id
			immortalModel.UpdateUserLevel(u.Id, uc.Level)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, msg)
		} else {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破失败,修为跌落`)
			randomLast := util.RandInt(int(math.Floor(float64(uc.Aura/3))), uc.Aura)
			// log.Println(randomLast)
			randomLast = int(math.Floor(float64(randomLast * (1 + (u.Lucky-10)/10))))
			// log.Println(randomLast)
			if randomLast > level.Up_need_aura {
				randomLast = level.Up_need_aura
			}
			send.SendGroupPost(msg["group_id"].(float64), `损失灵力`+Number2String(uc.Aura-randomLast))
			// log.Println(randomLast)
			// uc.Aura = randomLast
			immortalModel.UpdateUserAura(u.Id, randomLast-uc.Aura)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, msg)
		}

	}()

	return nil
}
func level3Up(u immortalModel.User, uc immortalModel.User_cultivate, level immortalModel.Level, msg map[string]interface{}) error {

	random := util.RandInt(1, 100)

	randomUp := float64(random * (1 + (u.Lucky-10)/10 + (u.Insight-10)/10))

	next_level, err := immortalModel.GetLevel(level.Next_level)
	if err != nil {
		return err
	}
	send.SendGroupPost(msg["group_id"].(float64), u.Name+`将开始突破到`+next_level.Name)
	go func() {
		// chatGPT.LevelUpBeforeStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2)
		// chatGPT.LevelUpIngStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2)
		if randomUp > 95 {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功`)
			uc.Level = next_level.Id
			immortalModel.UpdateUserLevel(u.Id, uc.Level)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, msg)
		} else {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破失败,修为跌落`)
			randomLast := util.RandInt(int(math.Floor(float64(uc.Aura/2))), uc.Aura)
			// log.Println(randomLast)
			randomLast = int(math.Floor(float64(randomLast * (1 + (u.Lucky-10)/10))))
			// log.Println(randomLast)
			if randomLast > level.Up_need_aura {
				randomLast = level.Up_need_aura
			}
			send.SendGroupPost(msg["group_id"].(float64), `损失灵力`+Number2String(uc.Aura-randomLast))
			// log.Println(randomLast)
			// uc.Aura = randomLast
			immortalModel.UpdateUserAura(u.Id, randomLast-uc.Aura)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, msg)
		}

	}()

	return nil
}
