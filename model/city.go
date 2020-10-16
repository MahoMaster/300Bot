package model

import (
	"fmt"
	"strings"
)

func GetCityId(name string) int {
	var cityId int = -1
	name = strings.TrimSpace(name)
	name = "%" + name + "%"
	err := db.Get(&cityId, `SELECT cid FROM city where area like ? limit 1`, name)
	if err != nil {
		fmt.Println(err)
	}
	return cityId
}
