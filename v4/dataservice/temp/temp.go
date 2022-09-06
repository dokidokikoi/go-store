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

// 将临时文件信息写入临时文件目录
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

// 临时文件转正，转正后将临时文件删除
func put(w http.ResponseWriter, r *http.Request) {
	uuid := strings.Split(r.URL.EscapedPath(), "/")[2]
	log.Println(uuid)

	// 读出临时文件信息
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

	// 获取临时文件的基本信息
	info, e := f.Stat()
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	actual := info.Size()
	// 将存有临时文件信息的文件删除
	os.Remove(infoFile)

	// 临时文件的大小与实际的大小不同，不能转正
	if actual != tempInfo.Size {
		os.Remove(datFile)
		log.Println("actual size mismatch, expect", tempInfo.Size, "actual", actual)
		return
	}
	commTempObject(datFile, tempInfo)
}

// 将临时文件移动到存储目录，并改名，加入缓存
func commTempObject(datFile string, tempInfo *tempInfo) {
	os.Rename(datFile, os.Getenv("STORE_ROOT")+"/object/"+tempInfo.Name)
	locate.Add(tempInfo.Name)
}

// 将临时文件存入临时目录
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

// 读取临时文件的信息文件
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

// 将临时文件的信息存储到临时目录并创建出临时文件的文件
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

	// 创建出临时文件的文件
	os.Create(os.Getenv("STORE_ROOT") + "/temp/" + t.Uuid + ".dat")

	w.Write([]byte(uuid))
}

// 删除临时文件
func del(w http.ResponseWriter, r *http.Request) {
	uuid := strings.Split(r.URL.EscapedPath(), "/")[2]
	infoFile := os.Getenv("STORE_ROOT") + "/temp/" + uuid
	datFile := infoFile + ".dat"
	os.Remove(infoFile)
	os.Remove(datFile)
}
