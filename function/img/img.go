package img

import (
	"300Bot/send"
	"300Bot/util"
	"fmt"
)

func SendOneImg(msg map[string]interface{}) {
	// rand.Seed(time.Now().Unix())
	random := fmt.Sprintf("%03d", util.RandInt(1, 640))
	img := `[CQ:image,file=http://img.mahomaster.com/blog/images/background/` + random + `.jpg]`
	send.SendGroupPost(msg["group_id"].(float64), img)
}
