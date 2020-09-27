package send

import (
	"300Bot/store"
	"fmt"

	"github.com/robfig/cron"
)

var tomorrow int64
var c *cron.Cron

func init() {
	c = cron.New()
	c.Start()
	//定义定时器
	timeInterval()
}

func timeInterval() {
	// 每天七点通知天气
	spec := "0 7 * * *"
	c.AddFunc(spec, func() {
		sendWether()
	})

	// sendWether()
}

func sendWether() {
	fmt.Println(store.GroupList)
	for _, value := range store.GroupList {
		SendGroup(value.Group_id, "测试7点准时定时发送消息")
	}
}
