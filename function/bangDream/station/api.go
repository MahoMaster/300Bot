package station

import (
	"300Bot/util"
	"encoding/json"
	"errors"
	"log"
	"time"
)

const host = "https://api.bandoristation.com"
const botName = "300Bot"
const stationToken = "nT8hqIcFgq"

func request(function string, data map[string]interface{}) []byte {
	data["function"] = function
	return util.HttpPost(host, data)
}

type Source struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type User struct {
	Type     string `json:"type"`
	User_id  int    `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type Room struct {
	Number      string `json:"number"`
	Raw_message string `json:"raw_message"`
	Source_info Source `json:"source_info"`
	Type        string `json:"type"`
	Time        int64  `json:"time"`
	User_info   User   `json:"user_info"`
}

type Res struct {
	Status   string      `json:"status"`
	Response interface{} `json:"response"`
}

func GetRoomList() []Room {
	var rooms = make([]Room, 0)
	var res Res
	json.Unmarshal(request("query_room_number", map[string]interface{}{
		"latest_time": time.Now().Add(-10*time.Minute).UnixNano() / int64(time.Millisecond),
	}), &res)
	log.Println("GetRoomList:", res)

	if res.Status == "success" {
		response := res.Response.([]interface{})
		for _, roomData := range response {
			roomBytes, _ := json.Marshal(roomData)
			var room Room
			json.Unmarshal(roomBytes, &room)
			rooms = append(rooms, room)
		}
	} else {
		log.Println("GetRoomList error:", res.Response.(string))
	}
	return rooms
}

func SubmitRoom(number string, qqStr string, msg string) error {
	var res Res
	json.Unmarshal(request("submit_room_number", map[string]interface{}{
		"number":      number,
		"user_id":     qqStr,
		"raw_message": msg,
		"source":      botName,
		"token":       stationToken,
	}), &res)
	log.Println("submitRoom:", res)
	if res.Status == "success" {
		return nil
	} else {
		return errors.New(res.Response.(string))
	}
}
