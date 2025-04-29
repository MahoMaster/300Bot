package present

import (
	"300Bot/send"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

func SendGift(qq string, msg map[string]interface{}) {
	rand.Seed(time.Now().Unix())
	random := rand.Intn(9)
	// qqStr := strconv.FormatFloat(qq, 'f', -1, 64)
	// fmt.Println(random)
	qq = strings.TrimSpace(qq)
	reg := regexp.MustCompile(`[1-9]([0-9]{5,11})`)
	if reg != nil {
		qqReg := reg.FindAllString(qq, -1)
		if len(qqReg) == 1 {
			qq = qqReg[0]
		}
	}
	qq = strings.TrimSpace(qq)
	send.SendGift(msg["group_id"].(float64), qq, random)
}
