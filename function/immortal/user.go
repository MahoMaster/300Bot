package immortal

import (
	"300Bot/function/chatGPT"
	"300Bot/model/immortalModel"
	"300Bot/send"
	"300Bot/util"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func CheckUserByQQ(qq string) (bool, bool) {
	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return false, true //不存在角色
	} else { //创建满10分钟可以删除
		now := int(time.Now().Unix())
		if now-u.Create_time > 300 {
			return true, true
		} else {
			return true, false
		}

	}
}
func CheckUserByName(name string) bool {
	_, err := immortalModel.GetUserInfoByName(name)
	if err != nil {
		return false
	} else {
		return true
	}
}
func DelUserByQQ(qq string) error {
	flag, flag2 := CheckUserByQQ(qq)
	if flag && flag2 {
		err := immortalModel.DelUserByQQ(qq)
		return err
	} else {
		if !flag {
			return errors.New("没账号")
		}
		if !flag2 {
			return errors.New("赶着投胎啊")
		}
	}
	return nil

}

func CreateUser(qq string, name string, msg map[string]interface{}) (immortalModel.User, error) {
	flag := CheckUserByName(name)
	if flag { //存在角色
		return immortalModel.User{}, errors.New("角色名重复")
	}

	rand.Seed(time.Now().UnixNano())

	user := immortalModel.User{}
	uc := immortalModel.User_cultivate{}
	user.Qq = qq
	user.Name = name
	user.Create_time = int(time.Now().Unix())
	send.SendGroupPost(msg["group_id"].(float64), "角色正在创建中")

	story := make([]string, 0)

	//随机灵根数,0.5%概率为单灵根，2%概率双灵根，5%概率三灵根，10%概率四灵根，100%概率五灵根
	roots := util.RandInt(0, 1000)
	rootsArr := []string{"Wood", "Gold", "Water", "Fire", "Earth"} //对应金木水火土
	rand.Shuffle(len(rootsArr), func(i, j int) {
		rootsArr[i], rootsArr[j] = rootsArr[j], rootsArr[i]
	})
	if roots < 5 {
		user.Roots_num = 1
		//单灵根比例 40 15 15 15 15
		user.SetValue(rootsArr[0], 40)
		user.SetValue(rootsArr[1], 15)
		user.SetValue(rootsArr[2], 15)
		user.SetValue(rootsArr[3], 15)
		user.SetValue(rootsArr[4], 15)

	} else if roots < 20 {
		user.Roots_num = 2
		//双灵根比例  32 32 12 12 12
		user.SetValue(rootsArr[0], 32)
		user.SetValue(rootsArr[1], 32)
		user.SetValue(rootsArr[2], 12)
		user.SetValue(rootsArr[3], 12)
		user.SetValue(rootsArr[4], 12)
	} else if roots < 50 {
		user.Roots_num = 3
		//三灵根比例  28 28 28 8 8
		user.SetValue(rootsArr[0], 28)
		user.SetValue(rootsArr[1], 28)
		user.SetValue(rootsArr[2], 28)
		user.SetValue(rootsArr[3], 8)
		user.SetValue(rootsArr[4], 8)
	} else if roots < 100 {
		user.Roots_num = 4
		//四灵根比例  23 23 23 23 8
		user.SetValue(rootsArr[0], 23)
		user.SetValue(rootsArr[1], 23)
		user.SetValue(rootsArr[2], 23)
		user.SetValue(rootsArr[3], 23)
		user.SetValue(rootsArr[4], 8)
	} else {
		user.Roots_num = 5
		//五灵根比例  20 20 20 20 20
		user.SetValue(rootsArr[0], 20)
		user.SetValue(rootsArr[1], 20)
		user.SetValue(rootsArr[2], 20)
		user.SetValue(rootsArr[3], 20)
		user.SetValue(rootsArr[4], 20)
	}

	//通报
	if user.Roots_num == 1 {
		story = append(story, "天选之人")
		send.SendGroupPost(msg["group_id"].(float64), "天选之人！单灵根！有欧皇！")
	}

	//随机数值
	user.Intelligence = util.RandInt(9, 12)
	user.Constitution = util.RandInt(9, 12)
	user.Insight = util.RandInt(9, 12)
	user.Spirit = util.RandInt(9, 12)
	user.Lucky = util.RandInt(8, 12)
	//通报
	if user.Intelligence == 12 && user.Constitution == 12 && user.Insight == 12 && user.Spirit == 12 {
		story = append(story, "筋骨奇佳")
		send.SendGroupPost(msg["group_id"].(float64), "筋骨奇佳！满资质！有欧皇！")

		//如果低于赋予三灵根
		if user.Roots_num > 3 {
			user.Roots_num = 3
			//三灵根比例  28 28 28 8 8
			user.SetValue(rootsArr[0], 28)
			user.SetValue(rootsArr[1], 28)
			user.SetValue(rootsArr[2], 28)
			user.SetValue(rootsArr[3], 8)
			user.SetValue(rootsArr[4], 8)
		}

	}
	//通报
	if user.Intelligence == 9 && user.Constitution == 9 && user.Insight == 9 && user.Spirit == 9 {
		story = append(story, "倒霉蛋")
		send.SendGroupPost(msg["group_id"].(float64), "天生残废！最低资质！有眉笔！奖励高幸运")

		//赋予高幸运
		user.Lucky = user.Lucky + 5
	}

	//设置凡人
	uc.Level = 1

	random := util.RandInt(0, 100)

	if random < 4 { //家财万贯 送1000灵石
		story = append(story, "家财万贯")
		uc.Stone = 1000
	}

	random = util.RandInt(0, 100)

	if random < 4 { //修仙世家 悟性+5，初始等级+1
		story = append(story, "修仙世家")
		uc.Level = 2
		user.Insight = user.Insight + 5
	}

	//创建角色
	err := immortalModel.CreateUser(user, uc)
	if err != nil {
		send.SendGroupPost(msg["group_id"].(float64), "但是创建角色报错了，让管理员看看吧")
		return user, err
	}

	storyStr := strings.Join(story, ",")

	if storyStr == "" {
		storyStr = "平凡人一个"
	}
	user.User_story = storyStr
	roots_num_str := RootsNum2RootsNumStr(user.Roots_num)
	//创建成功,发送属性
	tempalte := user.Name + `:
	体质:` + strconv.Itoa(user.Constitution) + `,
	智力:` + strconv.Itoa(user.Intelligence) + `,
	悟性:` + strconv.Itoa(user.Insight) + `,
	神识:` + strconv.Itoa(user.Spirit) + `,
	幸运:` + strconv.Itoa(user.Lucky) + `,
	` + roots_num_str + `，五行权重:
	金系:` + strconv.FormatFloat(user.Gold, 'f', -1, 64) + `,
	木系:` + strconv.FormatFloat(user.Wood, 'f', -1, 64) + `,
	水系:` + strconv.FormatFloat(user.Water, 'f', -1, 64) + `,
	火系:` + strconv.FormatFloat(user.Fire, 'f', -1, 64) + `
	土系:` + strconv.FormatFloat(user.Earth, 'f', -1, 64) + `
	背景:` + user.User_story
	send.SendGroupPost(msg["group_id"].(float64), tempalte)

	//发送背景故事
	chatGPT.GetUserStory(user.Name, storyStr+roots_num_str, qq, msg)
	return user, nil
}

func GetUserAllInfoByQQ(qq string, msg map[string]interface{}) error {
	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return err
	}
	uc, level, err := immortalModel.GetUserCultivateById(u.Id)
	if err != nil {
		return err
	}
	roots_num_str := RootsNum2RootsNumStr(u.Roots_num)
	tempalte := u.Name + `:
	体质:` + Number2String(u.Constitution) + `,
	智力:` + Number2String(u.Intelligence) + `,
	悟性:` + Number2String(u.Insight) + `,
	神识:` + Number2String(u.Spirit) + `,
	幸运:` + Number2String(u.Lucky) + `,
	` + roots_num_str + `，五行权重:
	金系:` + Number2String(u.Gold) + `,
	木系:` + Number2String(u.Wood) + `,
	水系:` + Number2String(u.Water) + `,
	火系:` + Number2String(u.Fire) + `
	土系:` + Number2String(u.Earth) + `
	背景:` + u.User_story + `
---------------------------
	` + level.Name + `:` + Number2String(uc.Aura) + `/` + Number2String(level.Up_need_aura) + `,
	灵石:` + Number2String(uc.Stone)

	send.SendGroupPost(msg["group_id"].(float64), tempalte)

	return nil

}