package objects

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"store/apiservice/heartbeat"
	"store/apiservice/locate"
	"store/apiservice/rs"
	"store/es"
	"store/utils"
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

	if m == http.MethodDelete {
		del(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func put(w http.ResponseWriter, r *http.Request) {
	hash := utils.GetHashFromHeader(r.Header)
	if hash == "" {
		log.Println("missing object hash in digest header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	size := utils.GetSizeFromHeader(r.Header)
	c, e := storeObject(r.Body, hash, size)
	if e != nil {
		log.Println(e)
		w.WriteHeader(c)
		return
	}
	if c != http.StatusOK {
		w.WriteHeader(c)
		return
	}

	name := strings.Split(r.URL.EscapedPath(), "/")[2]
	e = es.AddVersion(name, hash, size)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func storeObject(r io.Reader, hash string, size int64) (int, error) {
	if locate.Exist(utils.SetHash(hash)) {
		return http.StatusOK, nil
	}

	stream, e := putStream(utils.SetHash(hash), size)
	if e != nil {
		return http.StatusServiceUnavailable, e
	}

	reader := io.TeeReader(r, stream)
	d, e := utils.CalculateHash(reader)
	if e != nil {
		return http.StatusInternalServerError, e
	}

	if d != hash {
		stream.Commit(false)
		return http.StatusBadRequest, fmt.Errorf("object hash mismatch, calulated=%s, requested=%s", d, hash)
	}

	stream.Commit(true)

	return http.StatusOK, nil
}

func putStream(hash string, size int64) (*rs.RSPutStream, error) {
	servers := heartbeat.ChooseRandomDataServers(rs.ALL_SHARDS, nil)
	log.Println(servers)
	if len(servers) != rs.ALL_SHARDS {
		return nil, fmt.Errorf("cannot find enough dataServer")
	}

	return rs.NewRSPutStream(servers, hash, size)
}

func get(w http.ResponseWriter, r *http.Request) {
	name := strings.Split(r.URL.EscapedPath(), "/")[2]
	versionId := r.URL.Query()["version"]
	version := 0
	var e error

	if len(versionId) != 0 {
		version, e = strconv.Atoi(versionId[0])
		if e != nil {
			log.Println(e)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	meta, e := es.GetMetadata(name, version)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if meta.Hash == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	hash := utils.SetHash(meta.Hash)

	// size参数是因为RS码的实现要求每一个数据片的长度完全一样,
	// 在编码时如果对象长度不能被4整除,函数会对最后一个数据片进行填充。
	// 因此在解码时必须提供对象的准确长度,防止填充数据被当成原始对象数据返回。
	stream, e := getStream(hash, meta.Size)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, e = io.Copy(w, stream)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// GET 对象时会对缺失的分片进行即时修复,
	// 修复的过程也使用数据服务的 temp 接口,
	// RSGetStream 的 Close 方法用于在流关闭时将临时对象转正
	stream.Close()
}

func getStream(hash string, size int64) (*rs.RSGetStream, error) {
	locateinfo := locate.Locate(hash)
	if len(locateinfo) < rs.DATA_SHARDS {
		return nil, fmt.Errorf("object %s locate fail, result %v", hash, locateinfo)
	}

	dataServers := make([]string, 0)

	// locateinfo 的数量小于 ALL_SHARDS， 需要对缺失的分片修正
	if len(locateinfo) != rs.ALL_SHARDS {
		dataServers = heartbeat.ChooseRandomDataServers(rs.ALL_SHARDS-len(locateinfo), locateinfo)
	}

	return rs.NewRSGetStream(locateinfo, dataServers, hash, size)
}

func del(w http.ResponseWriter, r *http.Request) {
	name := strings.Split(r.URL.EscapedPath(), "/")[2]
	version, e := es.SearchLatestVersion(name)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	e = es.PutMetadata(name, version.Version+1, 0, "")
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
