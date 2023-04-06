package immortalModel

type Level struct {
	Id           int    `gorm:"primary_key" json:"id"` //
	Name         string `json:"name"`                  //名称
	Next_level   int    `json:"next_level"`            //下一个境界
	Up_need_aura int    `json:"up_need_aura"`          //突破需要灵力
}

type User struct {
	Id           int     `gorm:"primary_key" json:" - "` //
	Qq           string  `json:"qq"`                     //绑定qq
	Name         string  `json:"name"`                   //角色名
	Intelligence int     `json:"intelligence"`           //智力
	Constitution int     `json:"constitution"`           //体质
	Insight      int     `json:"insight"`                //悟性
	Spirit       int     `json:"spirit"`                 //神识
	Lucky        int     `json:"lucky"`                  //幸运
	Roots_num    int     `json:"roots_num"`              //灵根数，单灵根最优，五灵根最平凡
	Gold         float64 `json:"gold"`                   //金系权重
	Wood         float64 `json:"wood"`                   //木系权重
	Water        float64 `json:"water"`                  //水系权重
	Fire         float64 `json:"fire"`                   //火系权重
	Earth        float64 `json:"earth"`                  //土系权重
	User_story   string  `json:"user_story"`             //角色身份
	Is_delete    int     `json:"is_delete"`              //
	Is_ban       int     `json:"is_ban"`                 //
	Create_time  int     `json:"Create_time"`
}

func (user *User) SetValue(field string, value float64) {
	switch field {
	case "Gold":
		user.Gold = value
	case "Wood":
		user.Wood = value
	case "Water":
		user.Water = value
	case "Fire":
		user.Fire = value
	case "Earth":
		user.Earth = value
	}
}

type User_cultivate struct {
	Uid   int `gorm:"primary_key" json:"uid"` //
	Stone int `json:"stone"`                  //灵石数量
	Aura  int `json:"aura"`                   //灵气值
	Level int `json:"level"`                  //当前境界
}

type User_story struct {
	Uid         int    `gorm:"primary_key" json:"uid"` //
	Story       string `json:"story"`                  //
	Create_time int    `json:"create_time"`            //
}
