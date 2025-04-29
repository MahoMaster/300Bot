package immortalModel

// 默认从redis去读，读不到就赋初始值30点。last_time为从满点进入开始回复得时间，
type Action_Point struct {
	Point     int `json:"point"`
	Last_time int `json:"last_time"`
}

type Cultivate_Aura_Add struct {
	// Sum       int `json:"sum"`       //累计的数值
	Left_time  int     `json:"left_time"`  //剩余修炼时间
	Start_time int     `json:"start_time"` //刷新修炼的时间 前10分钟1.2倍速 前半小时满速 一小时70%  前三个小时50% 八个小时10%
	Count_time int     `json:"count_time"` //从start_time第几分钟已经清了sum
	Speed      float64 `json:"speed"`      //修炼速度
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
func (user *User) GetValue(field string) float64 {
	switch field {
	case "Gold":
		return user.Gold
	case "Wood":
		return user.Wood
	case "Water":
		return user.Water
	case "Fire":
		return user.Fire
	case "Earth":
		return user.Earth
	}
	return 0
}

type User_cultivate struct {
	Uid   int `gorm:"primary_key" json:"uid"` //
	Stone int `json:"stone"`                  //灵石数量
	Aura  int `json:"aura"`                   //灵气值
	Level int `json:"level"`                  //当前境界
}
type Level struct {
	Id           int    `gorm:"primary_key" json:"id"` //
	Name         string `json:"name"`                  //名称
	Next_level   int    `json:"next_level"`            //下一个境界
	Up_need_aura int    `json:"up_need_aura"`          //突破需要灵力
}

type User_story struct {
	Uid         int    `gorm:"primary_key" json:"uid"` //
	Story       string `json:"story"`                  //
	Create_time int    `json:"create_time"`            //
}

type Skill struct {
	Id       int           `gorm:"primary_key" json:"id"` //
	Name     string        `json:"name"`                  //
	Type     int           `json:"type"`                  //1为功法、2为技能
	Actived  int           `json:"actived"`               //1为主动技能，2为被动技能
	Level    int           `json:"level"`                 //等级
	Root     int           `json:"root"`                  //五行
	Intro    string        `json:"intro"`                 //描述
	Level_up int           `json:"level_up"`              //技能等级提升后对应的skill
	Entry    []Skill_entry `json:"entry" gorm:"-"`        //词条
}

type Skill_entry struct {
	Id      int     `gorm:"primary_key" json:"id"` //
	Sid     int     `json:"sid"`                   //
	Type    int     `json:"type"`                  //1为修炼类，2为真属性类，3为假属性类，4为伤害类
	Aim     string  `json:"aim"`                   //影响的值，例如aura,insight,hp,damage等
	Val     float64 `json:"val"`                   //具体影响数值
	Content string  `json:"content"`               //词条文本
}

type Equip struct {
	Id    int           `gorm:"primary_key" json:"id"` //
	Name  string        `json:"name"`                  //
	Level int           `json:"level"`                 //等级
	Root  int           `json:"root"`                  //五行
	Intro string        `json:"intro"`                 //描述
	Entry []Equip_entry `json:"entry" gorm:"-"`        //词条
	Type  int           `json:"type"`                  //1为丹炉
}

type Equip_entry struct {
	Id      int     `gorm:"primary_key" json:"id"` //
	Eid     int     `json:"eid"`                   //
	Type    int     `json:"type"`                  //1为修炼类，2为真属性类，3为假属性类，4为伤害类
	Aim     string  `json:"aim"`                   //影响的值，例如aura,insight,hp,damage等
	Val     float64 `json:"val"`                   //具体影响数值
	Content string  `json:"content"`               //词条文本
}
type Shop_admin struct {
	Id    int   `gorm:"primary_key" json:"id"` //
	Gid   int   `json:"gid"`                   //对应表得id
	Type  int   `json:"type"`                  //1为功法技能，2为装备，3为材料
	Price int   `json:"price"`                 //价格
	Skill Skill `json:"skill" gorm:"-"`        //查出来得技能信息
	Equip Equip `json:"equip" gorm:"-"`        //查出来得装备信息
}

type User_skill struct {
	Uid         int   `gorm:"primary_key" json:"uid"` //
	Sid         int   `json:"sid"`                    //技能id
	Is_equip    int   `json:"is_equip"`               //是否装备
	Create_time int   `json:"create_time"`            //
	Skill       Skill `json:"skill" gorm:"-"`         //查出来得技能信息
}

type User_equip struct {
	Uid         int   `gorm:"primary_key" json:"uid"` //
	Eid         int   `json:"eid"`                    //装备id
	Is_equip    int   `json:"is_equip"`               //是否装备
	Create_time int   `json:"create_time"`            //
	Equip       Equip `json:"equip" gorm:"-"`         //查出来得装备信息
}
