package store

import (
	"300Bot/model"
)

var GroupList []model.Group
var BanList []model.User

func init() {
	UpdateGroupList()
}

func UpdateGroupList() {
	GroupList = model.GetGroupList()
}

func UpdateBanList() {
	BanList = model.UpdateBanList()
}
