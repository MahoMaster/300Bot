package herosModel

import (
	"300Bot/function/heros/api"
)

func CheckSkill(skill api.Skill) bool {
	num := 0
	db.Get(&num, "select count(1) from skill where skill_id=?", skill.ID)
	if num != 0 {
		return true
	} else {
		url := "https://300report.jumpw.com/static/images/" + skill.IconFile
		db.Exec("insert into skill (name,icon,skill_id) values(?,?,?)", skill.Name, url, skill.ID)
		return true
	}
}
