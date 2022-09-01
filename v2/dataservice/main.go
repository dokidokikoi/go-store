package main

import (
	"log"
	"net/http"
	"os"

	"dataservice/heartbeat"
	"dataservice/locate"
	"dataservice/objects"
)

func main() {
	go heartbeat.StartHeartbeat()
	go locate.StartLocate()
	http.HandleFunc("/object/", objects.Handler)
	log.Fatal(http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil))
}
