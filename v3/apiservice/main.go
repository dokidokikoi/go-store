package main

import (
	"log"
	"net/http"
	"os"

	"store/apiservice/heartbeat"
	"store/apiservice/locate"
	"store/apiservice/objects"
	"store/apiservice/versions"
)

func main() {
	go heartbeat.ListenHeartbeat()
	http.HandleFunc("/object/", objects.Handler)
	http.HandleFunc("/locate/", locate.Handler)
	http.HandleFunc("/versions/", versions.Handler)
	log.Fatal(http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil))
}
