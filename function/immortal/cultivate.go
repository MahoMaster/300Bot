package immortal

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
	"errors"
	"fmt"
	"log"
	"time"
)

// 每次进入修炼会刷新时间累计
func Cultivate(qq string, use int, sid int, msg map[string]interface{}) error {

	err := immortalModel.UseActionPoint(qq, use)
	if err != nil {
		return err
	}

	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return err
	}
	_, _, caa, err := immortalModel.GetUserCultivateById(u.Id)
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
		speedUp = `主灵根加成:` + Number2String(self) + `%,
相生相克加成:` + fmt.Sprintf("%.2f", symbiosis/10-restrained/10) + `%`
	} else {
		speed = float64(speed * (1 + float64(u.Constitution-10)/10))
		speedUp = `体质加成:` + Number2String((u.Constitution-10)*10) + `%`
	}

	speed = float64(speed * (1 + float64(u.Insight-10)/10)) //悟性加成

	speed = speed / 30

	log.Println(speed)
	if caa.Left_time == 0 { //新建
		caa.Left_time = 20 * 60 * use
		caa.Speed = speed
		caa.Start_time = int(time.Now().Unix())
		caa.Count_time = 0
	} else {
		caa.Left_time = caa.Left_time + 20*60*use - caa.Count_time
		caa.Speed = speed
		caa.Start_time = int(time.Now().Unix())
		caa.Count_time = 0
	}

	err = immortalModel.SetUserCultivate(u.Id, caa)
	if err != nil {
		return err
	}

	template := `增加修炼时间` + Number2String(20*use) + `分钟,
` + speedUp + `,
悟性加成：` + Number2String((u.Insight-10)*10) + `%,
修炼速度:` + fmt.Sprintf("%.2f", speed) + `灵力/秒`

	send.SendGroupPost(msg["group_id"].(float64), template)

	return nil
}
