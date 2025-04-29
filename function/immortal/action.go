package immortal

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
	"300Bot/util"
	"errors"
	"math"
	"regexp"
	"time"
)

func Mining(qq string, msg map[string]interface{}, times int) error {
	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return errors.New("角色不存在")
	}
	err = immortalModel.UseActionPoint(qq, times)
	if err != nil {
		return err
	}
	stone := 0
	big := 0
	for i := 0; i < times; i++ {
		random := util.RandInt(3, 12)
		r := util.RandInt(1, 100)
		if r == 1 {
			random = 100
			big++
		}
		randomUp := float64(random * (1 + u.Constitution/100 + (u.Lucky-10)/100))
		stone = stone + int(math.Ceil(randomUp))
	}

	err = immortalModel.UpdateUserStone(u.Id, stone)
	if err != nil {
		return err
	}

	if big != 0 {
		send.SendGroupPost(msg["group_id"].(float64), "挖到了"+Number2String(big)+"个大灵石")
	}

	send.SendGroupPost(msg["group_id"].(float64), "挖矿"+Number2String(times)+"次成功，挖到了"+Number2String(stone)+"个灵石")

	return nil
}

func Steal(qq string, aimString string, msg map[string]interface{}) error {
	u, err := immortalModel.GetUserInfoByQQ(qq)
	if err != nil {
		return errors.New("角色不存在")
	}
	var aim immortalModel.User
	//解析对方qq
	aim, err = immortalModel.GetUserInfoByName(aimString)
	if err != nil {
		aim, err = immortalModel.GetUserInfoByQQ(aimString)
		if err != nil {
			re := regexp.MustCompile(`qq=(\d+)`)
			match := re.FindStringSubmatch(aimString)

			if len(match) > 1 {
				// fmt.Println("QQ:", )
				aim, err = immortalModel.GetUserInfoByQQ(match[1])
				if err != nil {
					return errors.New("目标不存在")
				}
			} else {
				return errors.New("目标不存在")
			}
		}
	}

	err = immortalModel.UseActionPoint(qq, 10)
	if err != nil {
		return err
	}

	//首先50%概率被发现
	random := util.RandInt(1, 100)
	uc, _, _, err := immortalModel.GetUserCultivateById(u.Id)
	if err != nil {
		return err
	}
	ac, _, _, err := immortalModel.GetUserCultivateById(aim.Id)
	if err != nil {
		return err
	}
	random = int(math.Floor(float64(random * (1 + ((ac.Level - uc.Level) * 10 / 100)))))   //修为差距
	random = int(math.Floor(float64(random * (1 + ((aim.Spirit - u.Spirit) * 10 / 100))))) //神识差距

	if random > 50 { //被发现
		send.SendGroupPost(msg["group_id"].(float64), u.Name+"被当场抓住，啥也没得到，还被打了一顿")
		last := int(math.Floor(float64(uc.Stone) * 0.03))
		if last != 0 {
			immortalModel.UpdateUserStone(aim.Id, last)
			immortalModel.UpdateUserStone(u.Id, -1*last)
		}

		send.SendGroupPost(msg["group_id"].(float64), u.Name+"掉落了"+Number2String(last)+"颗灵石，"+aim.Name+"捡起灵石，轻蔑一笑")

	} else {
		send.SendGroupPost(msg["group_id"].(float64), aim.Name+"完全没发现，"+u.Name+"就快得手了！")

		time.Sleep(1 * time.Second)

		all := int(math.Floor(float64(ac.Stone) * 0.1))
		steal := util.RandInt(0, all)

		steal = int(math.Floor(float64(steal) * ((float64(u.Lucky)-float64(aim.Lucky))/10 + 1)))

		if steal > all {
			steal = all
		}

		immortalModel.UpdateUserStone(u.Id, steal)
		immortalModel.UpdateUserStone(aim.Id, -1*steal)
		send.SendGroupPost(msg["group_id"].(float64), u.Name+"成功得手，获取了"+aim.Name+Number2String(steal)+"灵石")
	}

	// aim,err:=

	return nil
}
