package immortal

import (
	"300Bot/model/immortalModel"
	"300Bot/send"
	"300Bot/util"
	"errors"
	"math"
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
