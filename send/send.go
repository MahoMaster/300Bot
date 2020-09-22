package send

import (
	"300Bot/util"
)

var host = "http://127.0.0.1:5700"

func Send(msg string) {
	util.HttpGet(host + "/send_private_msg?user_id=675559614&message=" + msg)
}
