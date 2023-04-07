package immortalModel

import (
	"encoding/json"
	"errors"
	"math"
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
