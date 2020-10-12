package model

import (
	"strconv"
	"strings"
)

func SetImgBackground(qq float64, id string) bool {

	id = strings.TrimSpace(id)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return false
	}

	if idInt < 1 {
		return false
	}

	count := 0
	db.Get(&count, "select count(1) from imgbackground")
	if idInt > count {
		return false
	}

	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	num := 0
	db.Get(&num, "select count(1) from user where qq="+qqstr)
	if num != 0 {
		db.Exec("update user set imgbackground_set=" + id + " where qq=" + qqstr)
	} else {
		db.Exec("insert into user (qq,imgbackground_set) values('" + qqstr + "'," + id + ")")
	}
	return true
}
