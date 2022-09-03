package objects

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m == http.MethodPut {
		put(w, r)
		return
	}

	if m == http.MethodGet {
		get(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func put(w http.ResponseWriter, r *http.Request) {
	f, e := os.Create(os.Getenv("STORE_ROOT") + "/object/" + strings.Split(r.URL.EscapedPath(), "/")[2])
	if e != nil {
		log.Panicln(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer f.Close()

	io.Copy(f, r.Body)
}

func get(w http.ResponseWriter, r *http.Request) {
	f, e := os.Open(os.Getenv("STORE_ROOT") + "/object/" + strings.Split(r.URL.EscapedPath(), "/")[2])
	if e != nil {
		log.Panicln(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	io.Copy(w, f)
}
