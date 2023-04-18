package store

import (
	"300Bot/model"
	"300Bot/send"
	"encoding/json"
	"fmt"
)

var GroupList []model.Group
var BanList []model.User
var QQFriendList []model.QQFriend

func init() {
	UpdateGroupList()
	GetQQFriendList()
}

func UpdateGroupList() {
	GroupList = model.GetGroupList()
}

func UpdateBanList() {
	BanList = model.UpdateBanList()
}

func GetQQFriendList() {
	res := send.GetQQFriendList()
	var resp struct {
		Data    []model.QQFriend `json:"data"`
		RetCode int              `json:"retcode"`
		Status  string           `json:"status"`
	}
	json.Unmarshal(res, &resp)
	QQFriendList = resp.Data
}

func CheckQQFriend(qq string) bool {
	for _, f := range QQFriendList {
		if fmt.Sprintf("%f", f.User_id) == qq {
			return true
		}
	}
	return false
}
