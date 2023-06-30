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
	err := db.Get(&mods, "SELECT * from `user` where qq=?", qqstr)
	if err != nil {
		fmt.Println(err)
	}
	return mods
}

func UpdateBanList() []User {
	var mods = make([]User, 0)
	err := db.Select(&mods, "SELECT * from `user` where is_ban=1")
	if err != nil {
		fmt.Println(err)
	}
	return mods
}

func BanSomeOne(qqstr string, ban int) bool {
	// qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	num := 0
	db.Get(&num, "select count(1) from user where qq=?", qqstr)
	if num == 0 {
		return false
	}

	db.Exec("update user set is_ban=? where qq=?", ban, qqstr)
	return true
}

func GetChatGptInfo(qq float64) UserGPTSetting {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	var mod UserGPTSetting
	err := db.Get(&mod, "SELECT last_chatgpt,gpt_use_person,is_ban from `user` where qq=?", qqstr)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			CheckRegister(qqstr)
			// db.Exec("insert into user (qq) values(?)", qqstr)
		}
	}
	return mod
}

func LogUserUseTokens(qq float64, tokens_num int, last_id string) {

	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	CheckRegister(qqstr)
	_, err = db.Exec("update `user` set use_tokens=use_tokens+?,last_chatgpt=? where qq=?", tokens_num, last_id, qqstr)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func GetUserAllGPTPersonality() []GPTPersonality {
	var mods = make([]GPTPersonality, 0)
	err := db.Select(&mods, "SELECT qq as id,gpt_personality from `user` where not isNull(gpt_personality) and gpt_personality!=''")
	if err != nil {
		fmt.Println(err)
	}
	return mods
}

func SetGPTPersonality(qq float64, personality string) bool {
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)
	CheckRegister(qqstr)
	db.Exec("update user set gpt_personality=? where qq=?", personality, qqstr)
	return true
}
