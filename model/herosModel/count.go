package herosModel

import (
	"300Bot/function/heros/api"
)

func CountHerosWin(hero api.Hero, time string, matchID int, isWin int) {
	num := 0
	db.Get(&num, "select count(1) from count_heros_winrate where match_id=? and hero_id=?", matchID, hero.ID)
	if num != 0 {

	} else {
		db.Exec("insert into count_heros_winrate (hero_id,time,match_id,is_win) values(?,?,?,?)", hero.ID, time, matchID, isWin)

	}

}
