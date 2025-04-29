package immortalModel

func GetLevel(id int) (Level, error) {
	var level Level
	r := db.Table("level").Where("id=?", id).Limit(1).First(&level)
	return level, r.Error
}
