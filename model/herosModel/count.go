package herosModel

import (
	"300Bot/function/heros/api"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/garyburd/redigo/redis"
)

//
//英雄胜率统计部分
//
func CountHerosWin(hero api.Hero, time string, matchID int, isWin int) {
	num := 0
	db.Get(&num, "select count(1) from count_heros_winrate where match_id=? and heros_id=? and is_win=?", matchID, hero.ID, isWin)
	if num != 0 {

	} else {
		db.Exec("insert into count_heros_winrate (heros_id,time,match_id,is_win) values(?,?,?,?)", hero.ID, time, matchID, isWin)

	}

}

type HerosCount struct {
	Win  int     `json:"win,omitempty"`
	Lose int     `json:"lose,omitempty"`
	Name string  `json:"name,omitempty"`
	Icon string  `json:"icon,omitempty"`
	Rate float64 `json:"rate,omitempty"`
}

func DelHerosWinRedis() {
	c.Do("DEL", "300WinCount0desc")
	c.Do("DEL", "300WinCount1desc")
	c.Do("DEL", "300WinCount0asc")
	c.Do("DEL", "300WinCount1asc")
}

func GetHerosWin(time string, orderType int, descType string, limit string) []HerosCount {
	countList := make([]HerosCount, 0)
	limitInt, _ := strconv.Atoi(limit)
	redisKey := "300WinCount" + time + strconv.Itoa(orderType) + descType

	exit, err := redis.Bool(c.Do("EXISTS", redisKey))
	if err != nil {
		fmt.Println(err)
	} else {
		if exit {
			temp, _ := redis.String(c.Do("GET", redisKey))
			json.Unmarshal([]byte(temp), &countList)
			return countList[0:limitInt]
		}
	}

	useSql := ""
	if orderType == 0 {
		useSql = "( win + lose ) " + descType
	}
	if orderType == 1 {
		useSql = "rate " + descType
	}
	db.Select(&countList, "SELECT win,lose,w.`name`,win * 100 / ( win + lose ) AS rate,w.icon FROM(SELECT count( 1 ) AS win,c.heros_id,h.`name`,h.icon FROM `count_heros_winrate` AS c LEFT JOIN heros AS h ON h.heros_id = c.heros_id WHERE c.is_win = 1 AND time > ? GROUP BY heros_id ) AS w,(SELECT count( 1 ) AS lose,c.heros_id,h.`name` FROM `count_heros_winrate` AS c LEFT JOIN heros AS h ON h.heros_id = c.heros_id WHERE c.is_win = 0 AND time > ? GROUP BY heros_id ) AS l WHERE w.heros_id = l.heros_id ORDER BY "+useSql, time, time)
	temp, _ := json.Marshal(countList)
	c.Do("SET", redisKey, string(temp))
	c.Do("expire", redisKey, 86400)
	return countList[0:limitInt]
}

//
//装备胜率kd部分
//
type EquipCount struct {
	Equip_id int    `json:"equip_id,omitempty"`
	Name     string `json:"name,omitempty"`
	Time     string `json:"time"`
	Icon     string `json:"icon,omitempty"`
	Win      int    `json:"win,omitempty"`
	Lose     int    `json:"lose,omitempty"`
	Kill     int    `json:"kill,omitempty"`
	Death    int    `json:"death,omitempty"`
	Type     int    `json:"type"`
}

func CountEquipWinAndKd(count []EquipCount) {
	db.NamedExec("insert into count_equip_winrate (equip_id,time,win,lose,`kill`,death,type) values (:equip_id,:time,:win,:lose,:kill,:death,:type)", count)

}
