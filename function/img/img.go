package img

import (
	"300Bot/send"
	"fmt"
	"math/rand"
	"time"
)

func SendOneImg(msg map[string]interface{}) {
	rand.Seed(time.Now().Unix())
	random := fmt.Sprintf("%03d", rand.Intn(639)+1)
	img := `[CQ:image,file=http://img.mahomaster.com/blog/images/background/` + random + `.jpg]`
	send.SendGroupPost(msg["group_id"].(float64), img)
}
