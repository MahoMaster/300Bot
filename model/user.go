package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
	CheckRegister(qqstr)
	db.Exec("update user set imgbackground_set=? where qq=?", id, qqstr)
	return true
}

func CheckRegister(qqstr string) bool {
	num := 0
	db.Get(&num, "select count(1) from user where qq=?", qqstr)
	if num != 0 {
		return true
	} else {
		db.Exec("insert into user (qq) values(?)", qqstr)
		return true
	}
}

func CheckIn(qq float64) bool {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	CheckRegister(qqstr)

	var checkIn int32 = 0 //上一次打卡时间
	db.Get(&checkIn, "select check_in from user where qq=?", qqstr)

	t := time.Now()
	today := int32(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()) //凌晨时间

	if today > checkIn {
		now := int32(time.Now().Unix()) //当前时间
		db.Exec("update `user` set check_in=?,points=points+15 where qq=?", now, qqstr)
		return true
	} else {
		return false
	}
}

func GetUserInfo(qq float64) User {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	CheckRegister(qqstr)

	var mods User
	err = db.Get(&mods, "SELECT * from `user` where qq=?", qqstr)
	if err != nil {
		fmt.Println(err)
	}
	return mods
}
