package model

import (
	"strconv"
	"strings"
)

func SetImgBackground(qq float64, id string) {

	id = strings.TrimSpace(id)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return
	}

	if idInt < 1 {
		return
	}

	count := 0
	db.Get(&count, "select count(1) from imgbackground")
	if idInt > count {
		return
	}

	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	num := 0
	db.Get(&num, "select count(1) from user where qq="+qqstr)
	if num != 0 {
		db.Exec("update user set imgbackground_set=" + id + " where qq=" + qqstr)
	} else {
		db.Exec("insert into user (qq,imgbackground_set) values('" + qqstr + "'," + id + ")")
	}
}
