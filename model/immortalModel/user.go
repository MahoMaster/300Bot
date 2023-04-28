package immortalModel

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

func GetUserInfoByQQ(qq string) (User, error) {
	var user User
	r := db.Table("user").Where("qq=? and is_delete=0", qq).Limit(1).First(&user)
	if r.RowsAffected != 0 {
		return user, nil
	} else {
		return user, errors.New("没创建角色")
	}
}

func GetUserInfoByName(name string) (User, error) {
	var user User
	r := db.Table("user").Where("name=? and is_delete=0", name).Limit(1).First(&user)
	if r.RowsAffected != 0 {
		return user, nil
	} else {
		return user, errors.New("没创建角色")
	}
}

func CreateUser(user User, uc User_cultivate) error {
	result := db.Table("user").Create(&user)
	if result.Error != nil {
		return result.Error
	}
	uc.Uid = user.Id
	result = db.Table("user_cultivate").Create(&uc)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func DelUserByQQ(qq string) error {
	// result := db.Table("user").Save(&User{Qq: qq, Is_delete: 1})

	result := db.Table("user").Where("qq = ? and is_delete=0", qq).Update("is_delete", 1)
	// if result.Error != nil {
	return result.Error
	// }
	// return nil
}

func LogUserStoryByQQ(qq string, story string) error {
	u, err := GetUserInfoByQQ(qq)
	if err != nil {
		return err
	}
	result := db.Table("user_story").Create(&User_story{
		Uid:         u.Id,
		Story:       story,
		Create_time: int(time.Now().Unix()),
	})
	// if result.Error != nil {
	return result.Error
}

func GetUserCultivateById(uid int) (User_cultivate, Level, Cultivate_Aura_Add, error) {
	var uc User_cultivate
	var level Level
	var caa Cultivate_Aura_Add
	r := db.Table("user_cultivate").Where("uid=?", uid).Limit(1).First(&uc)
	if r.Error != nil {
		return uc, level, caa, r.Error
	}
	r = db.Table("level").Where("id=?", uc.Level).Limit(1).First(&level)

	caa, sum_aura, err := GetUserCultivateSum(uid)
	if err == nil {
		uc.Aura = uc.Aura + sum_aura
		if uc.Aura > level.Up_need_aura {
			uc.Aura = level.Up_need_aura
		}
		db.Table("user_cultivate").Where("uid=?", uid).Update("aura", uc.Aura)
	}

	return uc, level, caa, r.Error
}

func UpdateUserStone(uid int, stoneAdd int) error {
	result := db.Table("user_cultivate").Where("uid=?", uid).Update("stone", gorm.Expr("stone+?", stoneAdd))
	// if result.Error != nil {
	return result.Error
}

func UpdateUserAura(uid int, aura int) error {
	result := db.Table("user_cultivate").Where("uid=?", uid).Update("aura", gorm.Expr("aura+?", aura))
	// if result.Error != nil {
	return result.Error
}
func UpdateUserIntelligence(uid int, intelligence int, mode int) error {
	var update interface{}
	if mode == 1 {
		update = gorm.Expr("intelligence+?", intelligence)
	} else {
		update = intelligence
	}
	result := db.Table("user").Where("id=?", uid).Update("intelligence", update)
	// if result.Error != nil {
	return result.Error
}
func UpdateUserSpirit(uid int, spirit int, mode int) error {
	var update interface{}
	if mode == 1 {
		update = gorm.Expr("spirit+?", spirit)
	} else {
		update = spirit
	}
	result := db.Table("user").Where("id=?", uid).Update("spirit", update)
	// if result.Error != nil {
	return result.Error
}
func UpdateUserInsight(uid int, insight int, mode int) error {
	var update interface{}
	if mode == 1 {
		update = gorm.Expr("insight+?", insight)
	} else {
		update = insight
	}
	result := db.Table("user").Where("id=?", uid).Update("insight", update)
	// if result.Error != nil {
	return result.Error
}
func UpdateUserLevel(uid int, level int) error {
	result := db.Table("user_cultivate").Where("uid=?", uid).Update("level", level)
	// if result.Error != nil {
	return result.Error
}
func BuySkill(uid int, price int, sid int) error {
	err := db.Transaction(func(tx *gorm.DB) error {

		if err := tx.Table("user_cultivate").Where("uid=?", uid).Update("stone", gorm.Expr("stone-?", price)).Error; err != nil {
			return err
		}
		var us = User_skill{
			Uid:         uid,
			Sid:         sid,
			Is_equip:    0,
			Create_time: int(time.Now().Unix()),
		}
		if err := tx.Table("user_skill").Create(&us).Error; err != nil {
			return err
		}
		return nil
	})
	return err

}
func BuyEquip(uid int, price int, eid int) error {
	err := db.Transaction(func(tx *gorm.DB) error {

		if err := tx.Table("user_cultivate").Where("uid=?", uid).Update("stone", gorm.Expr("stone-?", price)).Error; err != nil {
			return err
		}
		var ue = User_equip{
			Uid:         uid,
			Eid:         eid,
			Is_equip:    0,
			Create_time: int(time.Now().Unix()),
		}
		if err := tx.Table("user_equip").Create(&ue).Error; err != nil {
			return err
		}
		return nil
	})
	return err

}
func GetUserSkillOne(uid int, sid int, hasDetail int) (User_skill, error) {
	var us User_skill

	r := db.Table("user_skill").Where("uid = ? and sid = ?", uid, sid).First(&us)
	if r.RowsAffected == 0 {
		return us, errors.New("未拥有该功法")
	}

	if hasDetail != 0 {
		skill, err := GetSkillDetail(sid, 1)
		if err != nil {
			return us, err
		}
		us.Skill = skill
	}

	return us, nil
}

func GetUserEquipOne(uid int, eid int, hasDetail int) (User_equip, error) {
	var ue User_equip

	r := db.Table("user_equip").Where("uid = ? and eid = ?", uid, eid).First(&ue)
	if r.RowsAffected == 0 {
		return ue, errors.New("未拥有该装备")
	}

	if hasDetail != 0 {
		equip, err := GetEquipDetail(eid, 1)
		if err != nil {
			return ue, err
		}
		ue.Equip = equip
	}

	return ue, nil
}

func GetUserSkillList(uid int, page int, is_equip int) ([]User_skill, error) {
	limit := 6
	start := (page - 1) * limit
	var us = make([]User_skill, 0)

	is_equipFilter := -1
	if is_equip == 0 {
		is_equipFilter = 1
	}
	if is_equip == 1 {
		is_equipFilter = 0
	}
	r := db.Table("user_skill").Where("uid = ? and is_equip!=?  limit ?,?", uid, is_equipFilter, start, limit).Find(&us)
	if r.RowsAffected == 0 {
		return us, errors.New("啥玩意儿啊，没有啊")
	}
	for index, item := range us {

		skill, err := GetSkillDetail(item.Sid, 0)
		if err != nil {
			return us, err
		}
		us[index].Skill = skill

	}
	return us, nil
}

func GetUserEquipList(uid int, page int, is_equip int) ([]User_equip, error) {
	limit := 6
	start := (page - 1) * limit
	var ue = make([]User_equip, 0)

	is_equipFilter := -1
	if is_equip == 0 {
		is_equipFilter = 1
	}
	if is_equip == 1 {
		is_equipFilter = 0
	}
	r := db.Table("user_equip").Where("uid = ? and is_equip!=?  limit ?,?", uid, is_equipFilter, start, limit).Find(&ue)
	if r.RowsAffected == 0 {
		return ue, errors.New("啥玩意儿啊，没有啊")
	}
	for index, item := range ue {

		equip, err := GetEquipDetail(item.Eid, 0)
		if err != nil {
			return ue, err
		}
		ue[index].Equip = equip

	}
	return ue, nil
}

func GetUserSkillEquipCount(uid int) (int64, error) {
	var count int64
	r := db.Table("user_skill").Where("uid = ? and is_equip!=?  ", uid, 1).Count(&count)
	return count, r.Error
}

func SetUserSkillEquip(us User_skill, u User) error {

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("user_skill").Where("uid=? and sid=?", us.Uid, us.Sid).Update("is_equip", us.Is_equip).Error; err != nil {
			return err
		}
		reviseFlag := false
		for _, entry := range us.Skill.Entry {
			if entry.Type == 2 {
				val := entry.Val
				if us.Is_equip == 0 {
					val = -1 * val
				}
				u.SetValue(entry.Aim, u.GetValue(entry.Aim)+val)
				reviseFlag = true
			}
		}
		if reviseFlag {
			if err := tx.Table("user").Where("id=?", u.Id).Save(&u).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
func SetUserEquipEquip(ue User_equip, u User) error {

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("user_equip").Where("uid=? and eid=?", ue.Uid, ue.Eid).Update("is_equip", ue.Is_equip).Error; err != nil {
			return err
		}
		reviseFlag := false
		for _, entry := range ue.Equip.Entry {
			if entry.Type == 2 {
				val := entry.Val
				if ue.Is_equip == 0 {
					val = -1 * val
				}
				u.SetValue(entry.Aim, u.GetValue(entry.Aim)+val)
				reviseFlag = true
			}
		}
		if reviseFlag {
			if err := tx.Table("user").Where("id=?", u.Id).Save(&u).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func GetUserEquipEquipType(uid int, equip_type int) (int64, error) {
	var count int64
	r := db.Table("user_equip as ue").Select("count(1)").Joins("left join equip as e on e.id=ue.eid").Where("uid = ? and is_equip=1 and e.type=?", uid, equip_type).Count(&count)
	return count, r.Error
}
func DelUserSkill(uid int, sid int) error {
	r := db.Table("user_skill").Where("uid = ? and sid = ?", uid, sid).Delete(User_skill{})
	return r.Error
}
