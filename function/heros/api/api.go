package api

import (
	"300Bot/util"
	"encoding/json"
	"strconv"
)

var host = "http://300report.jumpw.com"

// 获取玩家基本信息
type Role struct {
	RoleName   string // 角色名
	RoleLevel  int    // 角等级
	JumpValue  int    // 节操值
	WinCount   int    // 胜场数
	MatchCount int    // 总场数
	UpdateTime string // 更新日期
}
type UserRank struct {
	Type       int    // 排行类型
	RankName   string // 排行名称
	ValueName  string // 排行值名称
	Rank       int    // 排名
	Value      string // 排行值
	RankChange int    // 名次变化
}

type GetRoleRes struct {
	Result string
	Role   Role
	Rank   []UserRank
}

func Getrole(name string) (GetRoleRes, string) {
	res := util.HttpGet(host + "/api/getrole?name=" + name)
	var resp GetRoleRes
	json.Unmarshal(res, &resp)
	if resp.Result != "OK" {
		return resp, resp.Result
	} else {
		return resp, ""
	}
}

// 获取最新的战斗列表
type Hero struct {
	ID       int    // ID
	Name     string // 名称
	IconFile string // 图片相对路径.(在static/images/下)
}
type List struct {
	MatchID   int    // 比赛ID
	MatchType int    // 比赛类型(1:竟技场 2:战场)
	HeroLevel int    // 英雄等级
	Result    int    // 比赛结果(1:胜利 2:失败 3:逃跑)
	MatchDate string // 比赛日期
	Hero      Hero   // 英雄信息

}
type GetlistRes struct {
	Result string
	List   []List
}

func Getlist(name string, index string) (GetlistRes, string) {
	res := util.HttpGet(host + "/api/getlist?name=" + name + "&index=" + index)
	var resp GetlistRes
	json.Unmarshal(res, &resp)
	if resp.Result != "OK" {
		return resp, resp.Result
	} else {
		return resp, ""
	}
}

// 获取比赛详细信息
type Skill struct {
	ID       int    // ID
	Name     string // 名称
	IconFile string // 图片相对路径.(在static/images/下)
}
type Equip struct {
	ID       int    // ID
	Name     string // 名称
	IconFile string // 图片相对路径.(在static/images/下)
}
type EveryOne struct {
	RoleName      string  // 角色名
	RoleID        int     // 角色ID
	RoleLevel     int     // 角色等级
	HeroID        int     // 英雄ID
	HeroLevel     int     // 英雄等级
	Result        int     // 比赛结果(1:胜利 2:失败 3:逃跑)
	TeamResult    int     // 团队比赛结果(1:胜利 0:失败)
	IsFirstWin    int     // 是否首胜(1:首胜)
	KillCount     int     // 击杀数
	DeathCount    int     // 死亡数
	AssistCount   int     // 助攻数
	TowerDestroy  int     // 建筑摧毁数
	KillUnitCount int     // 小兵击杀数
	TotalMoney    int     // 金钱总数
	SkillID       []int   // 召唤师技能ID
	EquipID       []int   // 装备ID
	RewardMoney   int     // 金钱奖励
	RewardExp     int     // 经验奖励
	JumpValue     int     // 节操值
	WinCount      int     // 胜场数
	MatchCount    int     // 总场数
	ELO           int     // 团队(胜负)实力
	KDA           int     // 本场表现评分
	Hero          Hero    // 英雄信息
	Skill         []Skill // 召唤师技能信息
	Equip         []Equip // 装备信息
}
type Match struct {
	MatchType    int        // 比赛类型(1:竟技场 2:战场)
	WinSideKill  int        // 胜利方杀人数
	LoseSideKill int        // 失败方杀人数
	UsedTime     int        // 比赛所使用的时间(秒)
	MatchDate    string     // 比赛日期
	WinSide      []EveryOne // 胜利方角色信息
	LoseSide     []EveryOne // 失败方角色信息
}

type GetMatchRes struct {
	Result string
	Match  Match
}

func GetMatch(id int) (GetMatchRes, string) {
	res := util.HttpGet(host + "/api/getmatch?id=" + strconv.Itoa(id))
	var resp GetMatchRes
	json.Unmarshal(res, &resp)
	if resp.Result != "OK" {
		return resp, resp.Result
	} else {
		return resp, ""
	}
}

// 获取排行榜信息
type RankList struct {
	Index      int    // 名次
	Url        string // 链接地址
	Name       string // 玩家名
	Value      string // 值
	RankChange int    // 名次改变
}
type Rank struct {
	Title      string     // 标题
	IndexName  string     // 索引名称
	NameName   string     // 类型名称
	ValueName  string     // 值名称
	ChangeName string     // 变化名称
	List       []RankList // 排行榜
}
type GetRankRes struct {
	Result string
	Rank   Rank
}

func GetRank(typeId string, index string) (GetRankRes, string) {
	res := util.HttpGet(host + "/api/getrank?type=" + typeId + "&index=" + index)
	var resp GetRankRes
	json.Unmarshal(res, &resp)
	if resp.Result != "OK" {
		return resp, resp.Result
	} else {
		return resp, ""
	}
}

// func init() {
// 	Getrole("まほたん")
// 	// getrole("张三")
// }
