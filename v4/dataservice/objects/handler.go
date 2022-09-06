package objects

import (
	"io"
	"log"
	"net/http"
	"os"
	"store/utils"
	"strings"

	"store/dataservice/locate"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method

	if m == http.MethodGet {
		get(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func get(w http.ResponseWriter, r *http.Request) {
	file := getFile(strings.Split(r.URL.EscapedPath(), "/")[2])
	if file == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	sendFile(w, file)
}

func getFile(hash string) string {
	file := os.Getenv("STORE_ROOT") + "/object/" + hash
	f, _ := os.Open(file)
	h, e := utils.CalculateHash(f)
	if e != nil {
		log.Panicln(e)
		return ""
	}

	// 数据校验，因为可能由于硬件上的问题导致数据出错，例如数据降解
	d := utils.SetHash(h)
	f.Close()
	if d != hash {
		log.Println("object hash mismatch, remove", file)
		locate.Del(hash)
		os.Remove(file)
		return ""
	}

	return file
}

func sendFile(w io.Writer, file string) {
	f, _ := os.Open(file)
	defer f.Close()
	io.Copy(w, f)
}
