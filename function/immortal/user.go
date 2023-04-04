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

func CheckUserByQQ(qq string) bool {
	_, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return false //不存在角色
	} else {
		return true //存在角色
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

func CreateUser(qq string, name string, msg map[string]interface{}) (immortalModel.User, error) {
	flag := CheckUserByName(name)
	if flag { //存在角色
		return immortalModel.User{}, errors.New("角色名重复")
	}

	rand.Seed(time.Now().UnixNano())

	user := immortalModel.User{}
	uc := immortalModel.User_cultivate{}
	user.Qq = qq
	send.SendGroupPost(msg["group_id"].(float64), "角色正在创建中")

	story := make([]string, 0)

	//随机灵根数,0.5%概率为单灵根，2%概率双灵根，5%概率三灵根，10%概率四灵根，100%概率五灵根
	roots := util.RandInt(0, 1000)
	rootsArr := []string{"Wood", "Gold", "Water", "Fire", "Earth"} //对应金木水火土
	rand.Shuffle(len(rootsArr), func(i, j int) {
		rootsArr[i], rootsArr[j] = rootsArr[j], rootsArr[i]
	})
	roots_num_str := ""
	if roots < 5 {
		user.Roots_num = 1
		roots_num_str = "单灵根"
		//单灵根比例 40 15 15 15 15
		user.SetValue(rootsArr[0], 40)
		user.SetValue(rootsArr[1], 15)
		user.SetValue(rootsArr[2], 15)
		user.SetValue(rootsArr[3], 15)
		user.SetValue(rootsArr[4], 15)

	} else if roots < 20 {
		user.Roots_num = 2
		roots_num_str = "双灵根"
		//双灵根比例  32 32 12 12 12
		user.SetValue(rootsArr[0], 32)
		user.SetValue(rootsArr[1], 32)
		user.SetValue(rootsArr[2], 12)
		user.SetValue(rootsArr[3], 12)
		user.SetValue(rootsArr[4], 12)
	} else if roots < 50 {
		user.Roots_num = 3
		roots_num_str = "三灵根"
		//三灵根比例  28 28 28 8 8
		user.SetValue(rootsArr[0], 28)
		user.SetValue(rootsArr[1], 28)
		user.SetValue(rootsArr[2], 28)
		user.SetValue(rootsArr[3], 8)
		user.SetValue(rootsArr[4], 8)
	} else if roots < 100 {
		user.Roots_num = 4
		roots_num_str = "四灵根"
		//四灵根比例  23 23 23 23 8
		user.SetValue(rootsArr[0], 23)
		user.SetValue(rootsArr[1], 23)
		user.SetValue(rootsArr[2], 23)
		user.SetValue(rootsArr[3], 23)
		user.SetValue(rootsArr[4], 8)
	} else {
		user.Roots_num = 5
		roots_num_str = "五灵根"
		//五灵根比例  20 20 20 20 20
		user.SetValue(rootsArr[0], 23)
		user.SetValue(rootsArr[1], 23)
		user.SetValue(rootsArr[2], 23)
		user.SetValue(rootsArr[3], 23)
		user.SetValue(rootsArr[4], 8)
	}

	//通报
	if user.Roots_num == 1 {
		story = append(story, "天选之人的单灵根")
		send.SendGroupPost(msg["group_id"].(float64), "[CQ:at,qq=all] 天选之人！单灵根！有欧皇！")
	}

	//随机数值
	user.Intelligence = util.RandInt(9, 12)
	user.Constitution = util.RandInt(9, 12)
	user.Insight = util.RandInt(9, 12)
	user.Spirit = util.RandInt(9, 12)

	//通报
	if user.Intelligence == 12 && user.Constitution == 12 && user.Insight == 12 && user.Spirit == 12 {
		story = append(story, "筋骨奇佳的天之奇才")
		send.SendGroupPost(msg["group_id"].(float64), "[CQ:at,qq=all] 筋骨奇佳！满资质！有欧皇！")

		//如果低于赋予三灵根
		if user.Roots_num > 3 {
			user.Roots_num = 3
			roots_num_str = "三灵根"
			//三灵根比例  28 28 28 8 8
			user.SetValue(rootsArr[0], 28)
			user.SetValue(rootsArr[1], 28)
			user.SetValue(rootsArr[2], 28)
			user.SetValue(rootsArr[3], 8)
			user.SetValue(rootsArr[4], 8)
		}

	}

	random := util.RandInt(0, 100)

	if random < 4 { //家财万贯 送100灵石
		story = append(story, "家财万贯")
		send.SendGroupPost(msg["group_id"].(float64), "[CQ:at,qq=all] 家财万贯！有欧皇！")
		uc.Stone = 100
	}
	//设置凡人
	uc.Level = 1

	//创建角色
	err := immortalModel.CreateUser(user, uc)
	if err != nil {
		send.SendGroupPost(msg["group_id"].(float64), "但是创建角色报错了，让管理员看看吧")
		return user, err
	}

	storyStr := strings.Join(story, ",")
	//创建成功,发送属性
	tempalte := user.Name + `:
	体质:` + strconv.Itoa(user.Constitution) + `,
	智力:` + strconv.Itoa(user.Intelligence) + `,
	悟性:` + strconv.Itoa(user.Insight) + `,
	神识:` + strconv.Itoa(user.Spirit) + `,
	` + roots_num_str + `，五行权重:,
	金系:` + strconv.FormatFloat(user.Gold, 'f', -1, 64) + `,
	木系:` + strconv.FormatFloat(user.Wood, 'f', -1, 64) + `,
	水系:` + strconv.FormatFloat(user.Water, 'f', -1, 64) + `,
	火系:` + strconv.FormatFloat(user.Fire, 'f', -1, 64) + `
	土系:` + strconv.FormatFloat(user.Earth, 'f', -1, 64) + `
	背景:` + storyStr
	send.SendGroupPost(msg["group_id"].(float64), tempalte)

	//发送背景故事
	chatGPT.GetUserStory(user.Name, storyStr, qq, msg)
	return user, nil
}
