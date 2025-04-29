package herosModel

import (
	"300Bot/function/heros/api"
)

func CheckHero(hero api.Hero) bool {
	num := 0
	db.Get(&num, "select count(1) from heros where heros_id=?", hero.ID)
	if num != 0 {
		return true
	} else {
		url := "https://300report.jumpw.com/static/images/" + hero.IconFile
		db.Exec("insert into heros (name,icon,heros_id) values(?,?,?)", hero.Name, url, hero.ID)
		return true
	}
}
