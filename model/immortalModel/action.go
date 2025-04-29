package immortalModel

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"time"
)

// key为LumaImmortalActionPoints-qq
const keyF = "LumaImmortalActionPoints-"

func GetActionPoint(qq string) (Action_Point, error) {
	var ap Action_Point
	apStr, err := GetRedis(keyF + qq)
	if err != nil {
		if err.Error() != "key不存在" {
			return ap, err
		} else {
			ap.Point = 30
			ap.Last_time = int(time.Now().Unix())
			apStrB, err := json.Marshal(ap)
			if err != nil {
				return ap, err
			}
			apStr = string(apStrB)
		}
	}
	err = json.Unmarshal([]byte(apStr), &ap)
	if err != nil {
		return ap, err
	}

	//回复行动力
	now := int(time.Now().Unix())
	add := int(math.Floor(float64((now - ap.Last_time) / 600)))
	if ap.Point+add >= 30 {
		add = 30 - ap.Point
	}
	ap.Point = ap.Point + add
	ap.Last_time = add*600 + ap.Last_time
	return ap, nil
}

const keyC = "lumaImmortalCultivate-"

func GetUserCultivateSum(uid int) (Cultivate_Aura_Add, int, error) {
	uidStr := strconv.Itoa(uid)
	var caa Cultivate_Aura_Add
	var sum_aura int
	caaStr, err := GetRedis(keyC + uidStr)
	// log.Println(keyC + uidStr)
	// log.Println("这是查询到的数据", caaStr)
	if err != nil {
		return caa, sum_aura, err
	}
	err = json.Unmarshal([]byte(caaStr), &caa)
	if err != nil {
		return caa, sum_aura, err
	}

	//统计累计灵力
	now := int(time.Now().Unix())
	pass_time := now - caa.Start_time //距离开始过去了多少时间
	used := false
	if pass_time > caa.Left_time {
		pass_time = caa.Left_time
		used = true
	}
	has_count_time := caa.Count_time //已经统计了多少时间

	sum_time := caa.GetSumRank(pass_time) - caa.GetSumRank(has_count_time) //等价于这么多秒
	// log.Println("总共过去了多少时间", int(math.Floor((sum_time))))
	sum_aura = int(math.Floor((caa.Speed * sum_time)))
	caa.Count_time = pass_time

	if used {
		DelRedis(keyC + uidStr)
		caa.Left_time = 0
		caa.Count_time = 0
	} else {
		caaStrB, err := json.Marshal(caa)
		if err != nil {
			return caa, 0, err
		}
		// log.Println("这是计算后的数据", string(caaStrB))
		SetRedis(keyC+uidStr, string(caaStrB), 0)
	}
	return caa, sum_aura, nil
}

func SetUserCultivate(u User, caa Cultivate_Aura_Add, uc User_cultivate, level Level) (int, error) {
	uid := u.Id
	uidStr := strconv.Itoa(uid)
	caaStr, err := json.Marshal(caa)
	if err != nil {
		return 0, err
	}
	// log.Println("这是要设置的数据", string(caaStr))
	SetRedis(keyC+uidStr, string(caaStr), 0)
	needTime := caa.GetTimeFromAura(level.Up_need_aura-uc.Aura, caa.Speed)
	// log.Println(needTime)
	return needTime, nil
}

// 前10分钟1.2倍速 前半小时满速 一小时70%  前三个小时50% 八个小时10%
func (caa *Cultivate_Aura_Add) GetSumRank(timeInt int) float64 {
	// log.Println("输入的timeInt", timeInt)
	var sum float64 = 0
	time := float64(timeInt)
	if time <= 10*60 {
		sum = 1.2 * time
	} else if time < 30*60 {
		sum = 1.2*30*60 + 1*(time-10*60)
	} else if time < 60*60 {
		sum = 1.2*30*60 + 1*20*60 + 0.7*(time-30*60)
	} else if time < 3*60*60 {
		sum = 1.2*30*60 + 1*20*60 + 0.7*30*60 + 0.5*(time-60*60)
	} else if time < 8*60*60 {
		sum = 1.2*30*60 + 1*20*60 + 0.7*30*60 + 0.5*2*60*60 + 0.1*(time-3*60*60)
	} else {
		sum = 1.2*30*60 + 1*20*60 + 0.7*30*60 + 0.5*2*60*60 + 0.1*5*60*60
	}
	return sum
}

func (caa *Cultivate_Aura_Add) GetTimeFromAura(aura int, speed float64) int {
	time := 1 // 从第 1 秒开始计时
	for {
		// 计算当前时间的排名分数
		rank := caa.GetSumRank(time)
		// 计算当前时间的 aura 值
		currentAura := int(math.Floor(speed * rank))
		// 判断是否达到给定的 aura 值
		if currentAura >= aura {
			break
		}
		// 更新时间参数，继续计算
		time++
	}
	return time
}

func UseActionPoint(qq string, use int) error {
	// var ap Action_Point
	// apStr, err := GetRedis(keyF + qq)
	// if err != nil {
	// 	if err.Error() != "key不存在" {
	// 		return err
	// 	} else {
	// 		ap.Point = 30
	// 		ap.Last_time = int(time.Now().Unix())
	// 	}
	// } else {
	ap, err := GetActionPoint(qq)
	if err != nil {
		return err
	}

	//配置行动力回复
	if ap.Point == 30 {
		ap.Last_time = int(time.Now().Unix())
	}

	if ap.Point-use < 0 {
		return errors.New("行动力不足")
	}
	ap.Point = ap.Point - use

	apStrB, err := json.Marshal(ap)
	if err != nil {
		return err
	}
	SetRedis(keyF+qq, string(apStrB), 0)
	// }
	return nil
}
