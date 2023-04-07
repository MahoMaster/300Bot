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

	random := util.RandInt(3*times, 12*times)

	randomUp := float64(random * (1 + u.Constitution/100 + (u.Lucky-10)/100))

	stone := int(math.Ceil(randomUp))

	err = immortalModel.UpdateUserStone(u.Id, stone)
	if err != nil {
		return err
	}

	send.SendGroupPost(msg["group_id"].(float64), "挖矿"+Number2String(times)+"次成功，挖到了"+Number2String(stone)+"个灵石")

	return nil
}
