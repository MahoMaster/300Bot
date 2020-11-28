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
	if err := http.ListenAndServe(`:`+conf.Config.Port, nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
