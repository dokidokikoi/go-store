package temp

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"store/dataservice/locate"
	"store/utils"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m == http.MethodPut {
		put(w, r)
		return
	}
	if m == http.MethodPatch {
		patch(w, r)
		return
	}
	if m == http.MethodPost {
		post(w, r)
		return
	}
	if m == http.MethodDelete {
		del(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

type tempInfo struct {
	Uuid string
	Name string
	Size int64
}

func (t *tempInfo) writeToFile() error {
	f, e := os.Create(os.Getenv("STORE_ROOT") + "/temp/" + t.Uuid)
	if e != nil {
		return e
	}

	defer f.Close()

	b, _ := json.Marshal(t)
	f.Write(b)

	return nil
}

func (t *tempInfo) hash() string {
	s := strings.Split(t.Name, ".")

	return s[0]
}

func (t *tempInfo) id() int {
	s := strings.Split(t.Name, ".")
	id, _ := strconv.Atoi(s[1])

	return id
}

func put(w http.ResponseWriter, r *http.Request) {
	uuid := strings.Split(r.URL.EscapedPath(), "/")[2]
	log.Println(uuid)
	tempInfo, e := readFromFile(uuid)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	infoFile := os.Getenv("STORE_ROOT") + "/temp/" + uuid
	datFile := infoFile + ".dat"
	f, e := os.Open(datFile)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer f.Close()

	info, e := f.Stat()
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	actual := info.Size()
	os.Remove(infoFile)
	if actual != tempInfo.Size {
		os.Remove(datFile)
		log.Println("actual size mismatch, expect", tempInfo.Size, "actual", actual)
		return
	}
	commTempObject(datFile, tempInfo)
}

func commTempObject(datFile string, tempInfo *tempInfo) {
	f, _ := os.Open(datFile)
	d, e := utils.CalculateHash(f)
	if e != nil {
		log.Fatal(e)
	}

	d = utils.SetHash(d)
	f.Close()

	// /<hash>.X.<hash of shardX>
	os.Rename(datFile, os.Getenv("STORE_ROOT")+"/object/"+tempInfo.Name+"."+d)
	locate.Add(tempInfo.hash(), tempInfo.id())
}

func patch(w http.ResponseWriter, r *http.Request) {
	uuid := strings.Split(r.URL.EscapedPath(), "/")[2]
	tempinfo, e := readFromFile(uuid)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	infoFile := os.Getenv("STORE_ROOT") + "/temp/" + uuid
	datfile := infoFile + ".dat"
	f, e := os.OpenFile(datfile, os.O_WRONLY|os.O_APPEND, 0)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer f.Close()

	_, e = io.Copy(f, r.Body)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	info, e := f.Stat()
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	actual := info.Size()
	if actual > tempinfo.Size {
		os.Remove(datfile)
		os.Remove(infoFile)
		log.Println("actual size", actual, "exceeds", tempinfo.Size)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func readFromFile(uuid string) (*tempInfo, error) {
	f, e := os.Open(os.Getenv("STORE_ROOT") + "/temp/" + uuid)
	if e != nil {
		return nil, e
	}

	defer f.Close()

	b, _ := ioutil.ReadAll(f)
	var info tempInfo

	json.Unmarshal(b, &info)

	return &info, nil
}

func post(w http.ResponseWriter, r *http.Request) {
	output, _ := exec.Command("uuidgen").Output()
	uuid := strings.TrimSuffix(string(output), "\n")
	name := strings.Split(r.URL.EscapedPath(), "/")[2]
	size, e := strconv.ParseInt(r.Header.Get("size"), 0, 64)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t := tempInfo{
		Uuid: uuid,
		Name: name,
		Size: size,
	}
	e = t.writeToFile()
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	os.Create(os.Getenv("STORE_ROOT") + "/temp/" + t.Uuid + ".dat")
	w.Write([]byte(uuid))
}

func del(w http.ResponseWriter, r *http.Request) {
	uuid := strings.Split(r.URL.EscapedPath(), "/")[2]
	infoFile := os.Getenv("STORE_ROOT") + "/temp/" + uuid
	datFile := infoFile + ".dat"
	os.Remove(infoFile)
	os.Remove(datFile)
}
