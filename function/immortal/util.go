package immortal

import "strconv"

func RootsNum2RootsNumStr(roots_num int) string {
	switch roots_num {
	case 1:
		return "单灵根"
	case 2:
		return "双灵根"
	case 3:
		return "三灵根"
	case 4:
		return "四灵根"
	case 5:
		return "五灵根"
	default:
		return "五灵根"
	}
}

func Root2SkillRootName(root int) string {
	switch root {
	case 1:
		return "金系"
	case 2:
		return "木系"
	case 3:
		return "水系"
	case 4:
		return "火系"
	case 5:
		return "土系"
	default:
		return "无属性"
	}
}

func Root2SkillRootField(root int) string {
	switch root {
	case 1:
		return "Gold"
	case 2:
		return "Wood"
	case 3:
		return "Water"
	case 4:
		return "Fire"
	case 5:
		return "Earth"
	default:
		return "no"
	}
}

func GetSymbiosis(root int) []int { //被谁生  生谁
	switch root {
	case 1: // 土生金  金生水
		return []int{5, 3}
	case 2: // 水生木  木生火
		return []int{3, 4}
	case 3: // 金生水 水生木
		return []int{1, 2}
	case 4: // 木生火 火生土
		return []int{2, 5}
	case 5: //火生土 土生金
		return []int{4, 1}
	default:
		return []int{0, 0}
	}
}

func GetRestrained(root int) []int { //被i谁克，克谁  木剋土，土剋水，水剋火，火剋金，金剋木。
	switch root {
	case 1:
		return []int{4, 2}
	case 2:
		return []int{1, 5}
	case 3:
		return []int{5, 4}
	case 4:
		return []int{3, 1}
	case 5:
		return []int{2, 3}
	default:
		return []int{0, 0}
	}

}

func Type2SkillTypeName(typeInt int) string {
	switch typeInt {
	case 1:
		return "功法"
	case 2:
		return "秘技"
	default:
		return "功法"
	}
}

func Type2EquipTypeName(typeInt int) string {
	switch typeInt {
	case 1:
		return "丹炉"
	default:
		return "装备"
	}
}

func Number2String(num interface{}) string {
	if number, ok := num.(int); ok {
		return strconv.Itoa(number)
	}
	if number, ok := num.(float32); ok {
		return strconv.FormatFloat(float64(number), 'f', -1, 32)
	}
	if number, ok := num.(float64); ok {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return ""
}
