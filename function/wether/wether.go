package wether

import (
	"300Bot/model"
)

var groupList []model.Group

func init() {
	groupList = model.GetGroupList()
}

func SendWether() {
	// for key, value := range groupList {
	// 	// send.
	// }
}
