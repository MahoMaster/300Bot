package img

import (
	"300Bot/send"
	"fmt"
	"math/rand"
)

func SendOneImg(msg map[string]interface{}) {
	img := `[CQ:image,file=http://wmaho.xyz/blog/images/background/` + fmt.Sprintf("%03d", rand.Intn(639)+1) + `.jpg]`
	send.SendGroup(msg["group_id"].(float64), img)
}
