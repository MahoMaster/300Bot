package wether

import (
	"300Bot/conf"
	"300Bot/model"
	"300Bot/send"
	"encoding/json"
	"fmt"
	"log"

	"github.com/kirinlabs/HttpRequest"
)

func GetCityWether(name string, msg map[string]interface{}) {
	cityId := model.GetCityId(name)
	if cityId == -1 {
		send.SendGroupPost(msg["group_id"].(float64), "未查询到该地区")
		return
	} else {
		sendWetherData(cityId, msg)
	}

}

func sendWetherData(id int, msg map[string]interface{}) {

	url := "http://aliv18.data.moji.com/whapi/json/alicityweather/condition"
	req := HttpRequest.NewRequest()
	req.SetHeaders(map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": "APPCODE " + conf.Config.WetherApiCode,
	})

	res, err := req.Post(url, map[string]interface{}{
		"cityId": id,
	})
	if err != nil {
		// panic(err)
		log.Println(err)
	}
	defer res.Close()

	body, err := res.Body()
	if err != nil {
		fmt.Println(err)
		return
	}

	var resp map[string]interface{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp["code"].(float64) == 0 {
		condition := resp["data"].(map[string]interface{})["condition"].(map[string]interface{})
		city := resp["data"].(map[string]interface{})["city"].(map[string]interface{})

		tempalte := city["name"].(string) + `:` + condition["condition"].(string) + `,
湿度:` + condition["humidity"].(string) + `,
气压:` + condition["pressure"].(string) + `,
体感温度:` + condition["realFeel"].(string) + `,
测量温度:` + condition["temp"].(string) + `,
紫外线强度:` + condition["vis"].(string) + `,
风向:` + condition["windDir"].(string) + `,
风级:` + condition["windLevel"].(string) + `,
风速:` + condition["windSpeed"].(string) + `,
tips:` + condition["tips"].(string) + `
更新时间:` + condition["updatetime"].(string)
		send.SendGroupPost(msg["group_id"].(float64), tempalte)
	}
}
