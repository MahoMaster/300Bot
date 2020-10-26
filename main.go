package main

import (
	"300Bot/conf"
	"300Bot/controll"
	"fmt"
	"net/http"
	"os"
)

// var groupIdList []float64

func main() {
	// http.HandleFunc("/", controll.Index)
	// defer func() {
	// 	if info := recover(); info != nil {
	// 		fmt.Println("触发了宕机", info)
	// 	} else {
	// 		fmt.Println("芜锁胃")
	// 	}
	// }()
	controll.StartWebsocket()
	if err := http.ListenAndServe(`:`+conf.Config.Port, nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
