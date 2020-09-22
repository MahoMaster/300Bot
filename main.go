package main

import (
	"fmt"
	"net/http"
	"os"

	"300Bot/controll"
)

func main() {
	// http.HandleFunc("/", controll.Index)
	controll.StartWebsocket()

	if err := http.ListenAndServe(":9999", nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
