package immortalModel

import "errors"

func GetAdminShop(typeInt int, name string, page int) ([]Shop_admin, error) {
	var sa = make([]Shop_admin, 0)
	if page < 1 {
		page = 1
	}
	tableName := ""
	if typeInt == 1 {
		tableName = "skill"
	}
	start := (page - 1) * 5
	r := db.Table("shop_admin").Select("shop_admin.*").Joins("left join "+tableName+" on skill.id=shop_admin.gid").Where("shop_admin.type = ? and skill.name like ? limit ?,?", typeInt, "%"+name+"%", start, 10).Find(&sa)
	if r.RowsAffected == 0 {
		return sa, errors.New("没咯")
	}
	for index, item := range sa {

		if item.Type == 1 {
			var skill Skill
			r = db.Table("skill").Where("id = ?", item.Gid).First(&skill)
			if r.Error != nil {
				return sa, r.Error
			}
			sa[index].Skill = skill
		}
	}
	return sa, nil
}
