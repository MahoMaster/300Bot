package immortalModel

import (
	"errors"
	"time"
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
	result := db.Table("user_stroy").Create(&User_story{
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