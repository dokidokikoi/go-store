package objects

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

func getFile(name string) string {
	files, _ := filepath.Glob(os.Getenv("STORE_ROOT") + "/object/" + name + ".*")
	for len(files) != 1 {
		return ""
	}
	file := files[0]
	h := sha256.New()
	sendFile(h, file)
	d := utils.SetHash(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	fileHash := strings.Split(file, ".")[2]

	if d != fileHash {
		log.Println("object hash mismatch, renmove", file)
		locate.Del(fileHash)
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
