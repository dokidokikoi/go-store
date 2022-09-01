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
	go heartbeat.ListenHeartbeat()
	http.HandleFunc("/object/", objects.Handler)
	http.HandleFunc("/locate/", locate.Handler)
	log.Fatal(http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil))
}
