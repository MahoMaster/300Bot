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
