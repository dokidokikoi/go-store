package main

import (
	"log"
	"net/http"
	"os"

	"store/dataservice/heartbeat"
	"store/dataservice/locate"
	"store/dataservice/objects"
	"store/dataservice/temp"
)

func main() {
	locate.CollectObjects()
	go heartbeat.StartHeartbeat()
	go locate.StartLocate()
	http.HandleFunc("/object/", objects.Handler)
	http.HandleFunc("/temp/", temp.Handler)
	log.Fatal(http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil))
}
