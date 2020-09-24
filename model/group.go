package model

import (
	"fmt"
)

func GetGroupList() []Group {
	var mods = make([]Group, 0)
	err := db.Select(&mods, "SELECT id,group_id,manager from `group` where is_ban=0")
	if err != nil {
		fmt.Println(err)
	}
	return mods
}
