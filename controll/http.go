package controll

import (
	"300Bot/function/immortal"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func Test(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Fprintf(w, "hello")
}

func LumaCode2Info(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	mode := r.FormValue("mode")
	var res = make(map[string]interface{})
	res["code"] = 0
	info, err := immortal.Code2Info(code)
	if err != nil {
		res["msg"] = err.Error()
		res["code"] = 1
		res["data"] = nil
	} else {
		if mode == "1" {
			var infoMap = make(map[string]interface{})
			err = json.Unmarshal([]byte(info), &infoMap)
			if err != nil {
				res["msg"] = err.Error()
				res["code"] = 1
				res["data"] = nil
			} else {
				res["data"] = infoMap
			}
		}
	}
	resp, _ := json.Marshal(res)
	w.Write(resp)
}

func LumaReport(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	progress := r.FormValue("progress")
	log.Println(progress)
	mode := r.FormValue("mode")
	var res = make(map[string]interface{})
	res["code"] = 0

	immortal.BreakReport(code, progress, mode)
	resp, _ := json.Marshal(res)
	w.Write(resp)
}
