package immortal

import (
	"300Bot/conf"
	"300Bot/model/immortalModel"
	"300Bot/send"
	"errors"
	"fmt"
	"time"
)

// 突破
func Break(qq string, msg map[string]interface{}) error {
	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return err
	}
	uc, level, _, err := immortalModel.GetUserCultivateById(u.Id)
	if err != nil {
		return err
	}

	if uc.Aura < level.Up_need_aura {
		return errors.New("少年还需多加修炼")
	}

	switch level.Id {
	case 1:
		level1Up(u, uc, level, msg)
	case 2:
		level2Up(u, uc, level, msg)
	case 3:
		level3Up(u, uc, level, msg)
	case 4:
		level4Up(u, uc, level, msg)
	case 5:
		level5Up(u, uc, level, msg)
	case 6:
		// return errors.New("天地之间仿佛少了一些能够突破的规则，速速催促管理员写吧")
		level6Up(u, uc, level, msg)
	default:
		return errors.New("天地之间仿佛少了一些能够突破的规则，速速催促管理员写吧")
	}

	return nil
}

var logMoneyCultivate = make(map[string]*time.Timer)

// 每次进入修炼会刷新时间累计
func Cultivate(qq string, use int, sid int, msg map[string]interface{}) error {

	err := immortalModel.UseActionPoint(qq, use)
	if err != nil {
		return err
	}
	oneUse2Min := 10 //1行动点修炼10分钟
	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return err
	}
	uc, level, caa, err := immortalModel.GetUserCultivateById(u.Id)
	if err != nil {
		return err
	}
	us, err := immortalModel.GetUserSkillOne(u.Id, sid, 1)
	if err != nil {
		return err
	}
	var speed float64 = 0
	var root = 0
	for _, item := range us.Skill.Entry {
		if item.Type == 1 { //修炼类
			if item.Val > speed {
				speed = item.Val
				root = us.Skill.Root
			}
		}
	}

	if speed == 0 {
		return errors.New("都不是修炼功法你修炼个吊毛")
	}

	speedUp := ``
	//计算加成和速度
	if root != 0 {
		self := u.GetValue(Root2SkillRootField(root)) //同灵根值
		symbiosis := u.GetValue(Root2SkillRootField(GetSymbiosis(root)[0]))
		restrained := u.GetValue(Root2SkillRootField(GetRestrained(root)[0]))
		speed = float64(speed * (1 + self/100 + symbiosis/1000 - restrained/1000))
		speed = float64(speed * (1 + float64(u.Insight-10)/30)) //悟性加成
		speedUp = `主灵根加成:` + Number2String(self) + `%,
悟性加成：` + Number2String((u.Insight-10)*100/20) + `%,
相生相克加成:` + fmt.Sprintf("%.2f", symbiosis/10-restrained/10) + `%`
	} else {
		speed = float64(speed * (1 + float64(u.Constitution-10)*10/100))
		speedUp = `体质加成:` + Number2String((u.Constitution-10)*10) + `%`
	}

	speed = float64(speed * (1 + float64(uc.Level-1)*0.1)) //修为加成
	speedUp = speedUp + `
修为加成:` + Number2String(float64(uc.Level-1)*10) + `%`
	speed = speed / 60

	if caa.Left_time == 0 { //新建
		caa.Left_time = oneUse2Min * 60 * use
		caa.Speed = speed
		caa.Start_time = int(time.Now().Unix())
		caa.Count_time = 0
	} else {
		caa.Left_time = caa.Left_time + oneUse2Min*60*use - caa.Count_time
		caa.Speed = speed
		caa.Start_time = int(time.Now().Unix())
		caa.Count_time = 0
	}

	needTime, err := immortalModel.SetUserCultivate(u, caa, uc, level)
	if err != nil {
		return err
	}

	for _, moneyOne := range conf.Config.MoneyList {
		if moneyOne == qq {
			// log.Println(needTime)
			if needTime > 0 && caa.Left_time >= needTime {
				timer, ok := logMoneyCultivate[qq]
				if ok {
					if timer != nil {
						timer.Stop()
					}

				}

				timer = time.AfterFunc(time.Second*time.Duration(needTime), func() {
					send.SendGroupPost(msg["group_id"].(float64), "[CQ:at,qq="+qq+"] 你的修炼已经ok啦！")
					logMoneyCultivate[qq] = nil
				})
				logMoneyCultivate[qq] = timer

			}

		}
	}

	template := `增加修炼时间` + Number2String(oneUse2Min*use) + `分钟,
` + speedUp + `,
修炼速度:` + fmt.Sprintf("%.2f", speed*60) + `灵力/分钟,`

	send.SendGroupPost(msg["group_id"].(float64), template)

	return nil
}
