package ban

import (
	"300Bot/model"
	"300Bot/send"
	"300Bot/store"
	"regexp"
	"strings"
)

func BanSomeOne(qq string, ban int, msg map[string]interface{}) {
	qq = strings.TrimSpace(qq)
	reg := regexp.MustCompile(`[1-9]([0-9]{5,11})`)
	if reg != nil {
		qqReg := reg.FindAllString(qq, -1)
		if len(qqReg) == 1 {
			qq = qqReg[0]
		}
	}
	qq = strings.TrimSpace(qq)

	flag := model.BanSomeOne(qq, ban)
	if flag {
		if ban == 1 {
			send.SendGroupPost(msg["group_id"].(float64), "禁用成功")
		}
		if ban == 0 {
			send.SendGroupPost(msg["group_id"].(float64), "解除禁用成功")
		}
		store.UpdateBanList()
	} else {
		if ban == 1 {
			send.SendGroupPost(msg["group_id"].(float64), "禁用空气吗，爬")
		}
		if ban == 0 {
			send.SendGroupPost(msg["group_id"].(float64), "解除空气吗，爬")
		}
	}
}
