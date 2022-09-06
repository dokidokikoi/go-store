package main

import (
	"log"
	"net/http"
	"os"

	"apiservice/heartbeat"
	"apiservice/locate"
	"apiservice/objects"
)

func main() {
	// 监听心跳
	go heartbeat.ListenHeartbeat()
	http.HandleFunc("/object/", objects.Handler)
	http.HandleFunc("/locate/", locate.Handler)
	log.Fatal(http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil))
}
