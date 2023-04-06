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
