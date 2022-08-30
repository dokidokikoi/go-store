package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"demo/objects"
)

func main() {
	http.HandleFunc("/object/", objects.Handler)
	fmt.Println(os.Getenv("LISTEN_ADDRESS"))
	log.Fatal(http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil))
}
