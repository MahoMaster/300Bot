package store

import (
	"300Bot/model"
)

var GroupList []model.Group

func init() {
	UpdateGroupList()
}

func UpdateGroupList() {
	GroupList = model.GetGroupList()
}
