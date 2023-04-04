package immortalModel

import "errors"

func GetUserInfoByQQ(qq string) (User, error) {
	var user User
	r := db.Table("user").Where("qq=?", qq).Limit(1).First(&user)
	if r.RowsAffected != 0 {
		return user, nil
	} else {
		return user, errors.New("没创建角色")
	}
}

func GetUserInfoByName(name string) (User, error) {
	var user User
	r := db.Table("user").Where("name=?", name).Limit(1).First(&user)
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
