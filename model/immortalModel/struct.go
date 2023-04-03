package immortalModel

type Level struct {
	Id           int    `gorm:"primary_key" json:"id"` //
	Name         string `json:"name"`                  //名称
	Next_level   int    `json:"next_level"`            //下一个境界
	Up_need_aura int    `json:"up_need_aura"`          //突破需要灵力
}

type User struct {
	Id           int     `gorm:"primary_key" json:"id"` //
	Name         string  `json:"name"`                  //角色名
	Intelligence int     `json:"intelligence"`          //智力
	Constitution int     `json:"constitution"`          //体质
	Insight      int     `json:"insight"`               //悟性
	Spirit       int     `json:"spirit"`                //神识
	Roots_num    int     `json:"roots_num"`             //灵根数，单灵根最优，五灵根最平凡
	Gold         float32 `json:"gold"`                  //金系权重
	Wood         float32 `json:"wood"`                  //木系权重
	Water        float32 `json:"water"`                 //水系权重
	Fire         float32 `json:"fire"`                  //火系权重
	Earth        float32 `json:"earth"`                 //土系权重
}
type User_cultivate struct {
	Uid   int `gorm:"primary_key" json:"uid"` //
	Stone int `json:"stone"`                  //灵石数量
	Aura  int `json:"aura"`                   //灵气值
	Level int `json:"level"`                  //当前境界
}
