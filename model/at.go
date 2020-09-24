package model

import "fmt"

func GetAtList() []At {
	var mods = make([]At, 0)
	err := db.Select(&mods, "SELECT id,keyword,reply from `at` where is_delete=0")
	if err != nil {
		fmt.Println(err)
	}
	return mods
}
