package immortal

import (
	"300Bot/conf"
	"300Bot/function/chatGPT"
	"300Bot/model/immortalModel"
	"300Bot/send"
	"300Bot/store"
	"300Bot/util"
	"encoding/json"
	"errors"
	"math"
	"time"
)

// 奇奇怪怪的突破玩法的codekey，LumaImmortalUser2Code-code -> qq
const keyU = "LumaImmortalUser2Code-"

func SetAndSaveCodeInfo(mode int, info interface{}, msg map[string]interface{}) string {
	code := util.RandStr(8)
	if mode == 1 {
		var tmp = make(map[string]interface{})
		tmp["qq"] = info.(immortalModel.User).Qq
		tmp["name"] = info.(immortalModel.User).Name
		tmp["need_rank"] = 18
		tmp["msg"] = msg
		tmpB, _ := json.Marshal(tmp)
		immortalModel.SetRedis(keyU+code, string(tmpB), 1800)
	}

	return code
}

func Code2Info(code string) (string, error) {
	info, err := immortalModel.GetRedis(keyU + code)
	if err != nil {
		if err.Error() == "key不存在" {
			return "", errors.New("code不存在或已过期，请重新发起")
		} else {
			return "", err
		}
	}
	immortalModel.SetRedisExpire(keyU+code, 1800)
	return info, nil
}

var logBreakTimer = make(map[string]*time.Timer)

func BreakReport(code string, progress string, mode string) error {

	if mode == "1" {
		info, err := Code2Info(code)
		if err != nil {
			return err
		}
		var infoMap = make(map[string]interface{})
		err = json.Unmarshal([]byte(info), &infoMap)
		if err != nil {
			return err
		}
		msg := infoMap["msg"].(map[string]interface{})
		name := infoMap["name"].(string)
		qq := infoMap["qq"].(string)
		if progress != "100" && progress != "0" {
			send.SendGroupPost(msg["group_id"].(float64), name+`突破已突破`+progress+"%")
		}
		if progress == "100" {
			level4UpResult(qq, 1, code, msg)
		}
		if progress == "0" {
			level4UpResult(qq, 0, code, msg)
		}

	}

	return nil
}

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
		time.Sleep(2 * time.Second)
		// chatGPT.LevelUpIngStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2 * time.Second)
		if randomUp > 65 {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功，踏进了真正的修仙之路`)
			uc.Level = next_level.Id
			immortalModel.UpdateUserLevel(u.Id, uc.Level)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, "", msg)
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

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, "", msg)
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
		time.Sleep(2 * time.Second)
		// chatGPT.LevelUpIngStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2 * time.Second)
		if randomUp > 65 {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功`)
			uc.Level = next_level.Id
			immortalModel.UpdateUserLevel(u.Id, uc.Level)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, "", msg)
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

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, "", msg)
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
		time.Sleep(2 * time.Second)
		// chatGPT.LevelUpIngStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
		time.Sleep(2 * time.Second)
		if randomUp > 65 {
			send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功`)
			uc.Level = next_level.Id
			immortalModel.UpdateUserLevel(u.Id, uc.Level)

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, "", msg)
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

			chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, "", msg)
		}

	}()

	return nil
}
func level4UpResult(qq string, success int, code string, msg map[string]interface{}) error {
	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return err
	}
	uc, level, _, err := immortalModel.GetUserCultivateById(u.Id)
	if err != nil {
		return err
	}
	if level.Id != 4 {
		return errors.New("想干嘛！")
	}
	if uc.Aura < level.Up_need_aura {
		return errors.New("少年还需多加修炼")
	}
	next_level, err := immortalModel.GetLevel(level.Next_level)
	if err != nil {
		return err
	}
	immortalModel.DelRedis(keyU + code)
	if success == 1 {
		send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功`)
		uc.Level = next_level.Id
		immortalModel.UpdateUserLevel(u.Id, uc.Level)
		//智力+1
		intelligenceAdd := 1
		u.Intelligence = u.Intelligence + intelligenceAdd
		immortalModel.UpdateUserIntelligence(u.Id, intelligenceAdd, 1)
		send.SendGroupPost(msg["group_id"].(float64), u.Name+`智力+`+Number2String(intelligenceAdd))
		chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, "智力+1", msg)
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

		chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, "", msg)
	}

	return nil
}
func level4Up(u immortalModel.User, uc immortalModel.User_cultivate, level immortalModel.Level, msg map[string]interface{}) error {

	if !store.CheckQQFriend(u.Qq) {
		send.SendGroupPost(msg["group_id"].(float64), `先加我为QQ好友吧!`)
		return nil
	}

	code := SetAndSaveCodeInfo(1, u, msg)
	send.SendPrivatePostHasGroup(msg["user_id"].(float64), msg["group_id"].(float64), `您的突破限制:
http://`+conf.Config.Host+`/static/elsfk/index.html?code=`+code)

	send.SendGroupPost(msg["group_id"].(float64), `已经偷偷把突破条件发给你了，快去突破吧！`)
	// random := util.RandInt(1, 100)

	// randomUp := float64(random * (1 + (u.Lucky-10)/10 + (u.Insight-10)/10))

	// next_level, err := immortalModel.GetLevel(level.Next_level)
	// if err != nil {
	// 	return err
	// }
	// send.SendGroupPost(msg["group_id"].(float64), u.Name+`将开始突破到`+next_level.Name)
	// go func() {
	// 	// chatGPT.LevelUpBeforeStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
	// 	time.Sleep(2 * time.Second)
	// 	// chatGPT.LevelUpIngStory(u.Name, level.Name, next_level.Name, u.Qq, msg)
	// 	time.Sleep(2 * time.Second)
	// 	if randomUp > 95 {
	// 		send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破成功`)
	// 		uc.Level = next_level.Id
	// 		immortalModel.UpdateUserLevel(u.Id, uc.Level)

	// 		chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 1, msg)
	// 	} else {
	// 		send.SendGroupPost(msg["group_id"].(float64), u.Name+`突破失败,修为跌落`)
	// 		randomLast := util.RandInt(int(math.Floor(float64(uc.Aura/2))), uc.Aura)
	// 		// log.Println(randomLast)
	// 		randomLast = int(math.Floor(float64(randomLast * (1 + (u.Lucky-10)/10))))
	// 		// log.Println(randomLast)
	// 		if randomLast > level.Up_need_aura {
	// 			randomLast = level.Up_need_aura
	// 		}
	// 		send.SendGroupPost(msg["group_id"].(float64), `损失灵力`+Number2String(uc.Aura-randomLast))
	// 		// log.Println(randomLast)
	// 		// uc.Aura = randomLast
	// 		immortalModel.UpdateUserAura(u.Id, randomLast-uc.Aura)

	// 		chatGPT.LevelUpResultStory(u.Name, level.Name, next_level.Name, u.Qq, 0, msg)
	// 	}

	// }()

	return nil
}
