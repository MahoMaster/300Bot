package herosModel

import (
	"300Bot/function/heros/api"
)

func CheckEquip(equip api.Equip, equipType int) bool {
	num := 0
	db.Get(&num, "select count(1) from equip where equip_id=?", equip.ID)
	if num != 0 {
		return true
	} else {
		url := "https://300report.jumpw.com/static/images/" + equip.IconFile
		db.Exec("insert into equip (name,icon,equip_id,type) values(?,?,?,?)", equip.Name, url, equip.ID, equipType)
		return true
	}
}
