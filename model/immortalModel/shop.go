package immortalModel

import (
	"errors"
)

func GetAdminShop(typeInt int, name string, page int) ([]Shop_admin, error) {
	var sa = make([]Shop_admin, 0)
	if page < 1 {
		page = 1
	}
	tableName := ""
	if typeInt == 1 {
		tableName = "skill"
	}
	if typeInt == 3 {
		tableName = "equip"
	}
	limit := 6
	start := (page - 1) * limit
	r := db.Table("shop_admin").Select("shop_admin.*").Joins("left join "+tableName+" as g on g.id=shop_admin.gid").Where("shop_admin.type = ? and g.name like ? limit ?,?", typeInt, "%"+name+"%", start, limit).Find(&sa)
	if r.RowsAffected == 0 {
		return sa, errors.New("没咯")
	}
	for index, item := range sa {

		if item.Type == 1 {
			// var skill Skill
			// r = db.Table("skill").Where("id = ?", item.Gid).First(&skill)
			// if r.Error != nil {
			// 	return sa, r.Error
			// }
			skill, err := GetSkillDetail(item.Gid, 0)
			if err != nil {
				return sa, err
			}
			sa[index].Skill = skill
		}

		if item.Type == 3 {
			equip, err := GetEquipDetail(item.Gid, 0)
			if err != nil {
				return sa, err
			}
			sa[index].Equip = equip
		}
	}
	return sa, nil
}

func GetAdminShopOne(typeInt int, gid int, hasDetail int) (Shop_admin, error) {
	var sa Shop_admin
	r := db.Table("shop_admin").Where("type = ? and gid=?", typeInt, gid).First(&sa)
	if r.Error != nil {
		return sa, r.Error
	}
	if hasDetail != 0 {
		if sa.Type == 1 {
			// r = db.Table("skill").Where("id = ?", sa.Gid).First(&skill)
			// if r.Error != nil {
			// 	return sa, r.Error
			// }
			skill, err := GetSkillDetail(sa.Gid, 0)
			if err != nil {
				return sa, err
			}
			sa.Skill = skill
		}
		if sa.Type == 3 {
			equip, err := GetEquipDetail(sa.Gid, 0)
			if err != nil {
				return sa, err
			}
			sa.Equip = equip
		}
	}
	return sa, nil
}
