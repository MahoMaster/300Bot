package immortalModel

func GetSkillDetail(sid int, hasDetail int) (Skill, error) {
	var skill Skill
	r := db.Table("skill").Where("id=?", sid).First(&skill)
	if r.Error != nil {
		return skill, r.Error
	}
	if hasDetail != 0 {
		var entry = make([]Skill_entry, 0)
		r = db.Table("skill_entry").Where("sid=?", sid).Find(&entry)
		if r.Error != nil {
			return skill, r.Error
		}
		skill.Entry = entry
	}

	return skill, nil
}
