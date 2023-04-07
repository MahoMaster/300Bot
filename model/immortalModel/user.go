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

func GetUserCultivateById(uid int) (User_cultivate, Level, error) {
	var uc User_cultivate
	var level Level
	r := db.Table("user_cultivate").Where("uid=?", uid).Limit(1).First(&uc)
	if r.Error != nil {
		return uc, level, r.Error
	}
	r = db.Table("level").Where("id=?", uc.Level).Limit(1).First(&level)
	return uc, level, r.Error
}

func UpdateUserStone(uid int, stoneAdd int) error {
	result := db.Table("user_cultivate").Where("uid=?", uid).Update("stone", gorm.Expr("stone+?", stoneAdd))
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

func GetUserSkill(uid int, sid int, hasDetail int) (User_skill, error) {
	var us User_skill

	r := db.Table("user_skill").Where("uid = ? and sid = ?", uid, sid).First(&us)
	if r.RowsAffected == 0 {
		return us, errors.New("不存在")
	}

	if hasDetail != 0 {
		skill, err := GetSkillDetail(sid, 0)
		if err != nil {
			return us, err
		}
		us.Skill = skill
	}

	return us, nil
}
