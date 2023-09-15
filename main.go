package main

import (
	"300Bot/conf"
	"300Bot/controll"
	_ "300Bot/interval"
	"fmt"
	"net/http"
	"os"
)

func main() {
	controll.StartWebsocket()
	http.HandleFunc("/lumaCode2Info", controll.LumaCode2Info)
	http.HandleFunc("/lumaReport", controll.LumaReport)
	http.HandleFunc("/SendMeQQ", controll.SendMeQQ)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	if err := http.ListenAndServe(`:`+conf.Config.Port, nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
