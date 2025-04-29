package heros

import (
	"300Bot/function/heros/api"
	"300Bot/model/herosModel"
	"300Bot/util"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

var removeRepetitio sync.Map
var lostRankListNum = 0
var lostMatchListNum = 0
var lostMatchInfoNum = 0

func GetDailyData() {
	log.Println("开始获取数据")
	begin := time.Now()
	//获取每天团分前2000人的数据
	rankList := []api.RankList{}
	for i := 0; i < 2000; i += 50 {
		temp, msg := api.GetRank("1", strconv.Itoa(i))
		if msg == "" {
			rankList = append(rankList, temp.Rank.List...)
		} else {
			lostRankListNum++
		}

	}
	log.Printf("已获取共%d个人数据", len(rankList))

	g := goroutineNew(6)

	//获取每个人今天打的竞技场数据
	matchList := []api.List{}

	t := time.Now()
	today := int64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()) - 60*60*24 //凌晨时间
	todayStr := util.Time2Str(today)
	for _, value := range rankList {
		g.goroutineRun(func() {
			getOneMatchList(value, todayStr, &matchList)
		})
	}
	wg.Wait()
	log.Printf("已获取到所有人的今日比赛数据,共%d条", len(matchList))
	//收集比赛数据
	log.Println("开始处理数据")

	var equipData sync.Map //统计装备胜负kd
	for _, value := range matchList {
		g.goroutineRun(func() {
			getMatchData(value, &equipData)
		})
	}
	wg.Wait()

	//装备数据转化为数组存入数据库
	equipCount := make([]herosModel.EquipCount, 0)
	equipData.Range(func(k, v interface{}) bool {
		equipCount = append(equipCount, v.(herosModel.EquipCount))
		return true
	})
	herosModel.CountEquipWinAndKd(equipCount)
	end := time.Now()
	//删除redis缓存数据
	herosModel.DelHerosWinRedis()
	log.Println("今日数据收集已完毕")
	fmt.Println("丢失人物列表次数:", lostRankListNum)
	fmt.Println("丢失人物列表次数:", lostMatchListNum)
	fmt.Println("丢失人物列表次数:", lostMatchInfoNum)
	fmt.Println("总共用时:", end.Sub(begin))

	//重置部分数据
	removeRepetitio = sync.Map{}
	lostRankListNum = 0
	lostMatchListNum = 0
	lostMatchInfoNum = 0
}

//获取每个人今日打了哪些场
func getOneMatchList(value api.RankList, todayStr string, matchList *[]api.List) {
	oneMatchList := []api.List{}
	i := 0
	for true {
		temp, msg := api.Getlist(value.Name, strconv.Itoa(i))
		if msg == "" {
			if len(temp.List) != 0 {
				time := temp.List[0].MatchDate
				if time >= todayStr {
					oneMatchList = append(oneMatchList, temp.List...)
					i += 10
				} else {
					break
				}
			} else {
				break
			}
		} else {
			lostMatchListNum++
			break
		}
	}
	*matchList = append(*matchList, oneMatchList...)
	wg.Done()
	// log.Println("完成1人")
}

//获取比赛详细数据
func getMatchData(match api.List, equipData *sync.Map) {
	matchInfo, msg := api.GetMatch(match.MatchID)
	if msg == "" {
		_, hasSolved := removeRepetitio.Load(match.MatchID)
		if hasSolved {
			wg.Done()
			return
		} else {
			removeRepetitio.Store(match.MatchID, 1)
		}
		// fmt.Println(matchInfo)
		for _, value := range matchInfo.Match.WinSide {
			getSkillEquipHeros(value, matchInfo.Match.MatchType, equipData, 1)
			if matchInfo.Match.MatchType == 1 {
				herosModel.CountHerosWin(value.Hero, matchInfo.Match.MatchDate, match.MatchID, 1)
			}
		}
		for _, value := range matchInfo.Match.LoseSide {
			getSkillEquipHeros(value, matchInfo.Match.MatchType, equipData, 0)
			if matchInfo.Match.MatchType == 1 {
				herosModel.CountHerosWin(value.Hero, matchInfo.Match.MatchDate, match.MatchID, 0)
			}
		}
	} else {
		lostMatchInfoNum++
	}
	wg.Done()
	// log.Println("完成1局")
}

func getSkillEquipHeros(one api.EveryOne, matchType int, equipData *sync.Map, is_win int) {
	//处理技能
	for _, value := range one.Skill {
		herosModel.CheckSkill(value)
	}
	//处理Hero

	herosModel.CheckHero(one.Hero)

	//处理装备
	for _, value := range one.Equip {
		if matchType == 2 {
			//战场模式
			herosModel.CheckEquip(value, 1)
			getEquipData(equipData, one, value, is_win, 1)
		} else {
			herosModel.CheckEquip(value, 0)
			getEquipData(equipData, one, value, is_win, 0)
		}
	}
}

//统计装备kd胜负
func getEquipData(equipData *sync.Map, one api.EveryOne, equip api.Equip, is_win int, matchType int) {
	count1, has := equipData.Load(strconv.Itoa(matchType) + ":" + strconv.Itoa(equip.ID))
	win := 0
	lose := 0
	if is_win == 1 {
		win = 1
		lose = 0
	} else {
		win = 0
		lose = 1
	}
	if !has {
		t := time.Now()
		today := int64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()) - 60*60*24 //凌晨时间
		todayStr := util.Time2Str(today)
		url := "https://300report.jumpw.com/static/images/" + equip.IconFile

		count := herosModel.EquipCount{equip.ID, equip.Name, todayStr, url, win, lose, one.KillCount, one.DeathCount, matchType}
		equipData.Store(strconv.Itoa(matchType)+":"+strconv.Itoa(equip.ID), count)
	} else {
		count := count1.(herosModel.EquipCount)
		count.Kill = count.Kill + one.KillCount
		count.Death = count.Death + one.DeathCount
		count.Win = count.Win + win
		count.Lose = count.Lose + lose
		equipData.Store(strconv.Itoa(matchType)+":"+strconv.Itoa(equip.ID), count)
	}

}
