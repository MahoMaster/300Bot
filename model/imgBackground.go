package model

import (
	"fmt"
	"strconv"
)

func GetImgBackGroundInfo(qq float64) ImgBackground {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	id := 1
	err := db.Get(&id, "SELECT imgbackground_set from `user` where qq=?", qqstr)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			CheckRegister(qqstr)
			// db.Exec("insert into user (qq) values(?)", qqstr)
		}
	}
	var mods ImgBackground
	err = db.Get(&mods, "SELECT * from `imgbackground` where id=?", id)
	if err != nil {
		fmt.Println(err)
	}
	return mods
}
