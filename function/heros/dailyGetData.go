package heros

import (
	"300Bot/function/heros/api"
	"300Bot/model/herosModel"
	"300Bot/util"
	"fmt"
	"strconv"
	"time"
)

func GetDailyData() {
	//获取每天团分前2000人的数据
	rankList := []api.RankList{}
	for i := 0; i < 2000; i += 50 {
		temp, msg := api.GetRank("1", strconv.Itoa(i))
		if msg == "" {
			rankList = append(rankList, temp.Rank.List...)
		}

	}
	fmt.Println("已获取前2000人个人数据")

	//获取每个人今天打的竞技场数据
	matchList := []api.List{}

	t := time.Now()
	today := int64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()) - 60*60*24 //凌晨时间
	todayStr := util.Time2Str(today)
	for _, value := range rankList {
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
				break
			}
		}
		matchList = append(matchList, oneMatchList...)
	}
	fmt.Println("已获取到所有人的今日比赛数据")

	//收集比赛数据
	fmt.Println("开始处理数据")
	for _, value := range matchList {
		getMatchData(value)
	}
	herosModel.DelHerosWinRedis()
	fmt.Println("今日数据收集已完毕")
}

func getMatchData(match api.List) {
	matchInfo, msg := api.GetMatch(match.MatchID)
	if msg == "" {
		// fmt.Println(matchInfo)
		for _, value := range matchInfo.Match.WinSide {
			getSkillEquipHeros(value, matchInfo.Match.MatchType)
			if matchInfo.Match.MatchType == 1 {
				herosModel.CountHerosWin(value.Hero, matchInfo.Match.MatchDate, match.MatchID, 1)
			}
		}
		for _, value := range matchInfo.Match.LoseSide {
			getSkillEquipHeros(value, matchInfo.Match.MatchType)
			if matchInfo.Match.MatchType == 1 {
				herosModel.CountHerosWin(value.Hero, matchInfo.Match.MatchDate, match.MatchID, 0)
			}
		}
	}
}

func getSkillEquipHeros(one api.EveryOne, matchType int) {
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
		} else {
			herosModel.CheckEquip(value, 0)
		}
	}
}
