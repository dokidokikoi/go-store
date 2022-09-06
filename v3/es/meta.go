package es

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// 文件元数据
type Metadata struct {
	Name    string
	Version int
	Size    int64
	Hash    string
}

// 通过 name 和 version 获取元数据
func getMetadata(name string, versionId int) (meta Metadata, e error) {
	url := fmt.Sprintf("http://%s/metadata/_doc/%s_%d/_source", os.Getenv("ES_SERVER"), name, versionId)

	r, e := http.Get(url)
	if e != nil {
		return
	}
	if r.StatusCode != http.StatusOK {
		e = fmt.Errorf("fail to get %s_%d: %d", name, versionId, r.StatusCode)
		return
	}

	result, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(result, &meta)

	return
}

// 获取 name 最新的元数据
func SearchLatestVersion(name string) (meta Metadata, e error) {
	doc := fmt.Sprintf(`{"query":{"match":{"name":"%s"}},"sort": [{"version":{"order":"desc"}}],"from":0,"size":1}`, name)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/metadata/_search", os.Getenv("ES_SERVER"))
	log.Println(url)
	log.Println(doc)
	request, _ := http.NewRequest("POST", url, strings.NewReader(doc))
	request.Header.Set("Content-Type", "application/json")
	r, e := client.Do(request)
	if e != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		e = fmt.Errorf("fail to search latest metadata: %d", r.StatusCode)
		return
	}

	result, _ := ioutil.ReadAll(r.Body)

	var sr searchResult
	json.Unmarshal(result, &sr)
	if len(sr.Hits.Hits) != 0 {
		meta = sr.Hits.Hits[0].Source
	}

	return
}

// 如果版本为0，获取最新元数据
func GetMetadata(name string, version int) (meta Metadata, e error) {
	if version == 0 {
		return SearchLatestVersion(name)
	}

	return getMetadata(name, version)
}

// 添加元数据，指定版本
func PutMetadata(name string, version int, size int64, hash string) error {
	doc := fmt.Sprintf(`{"name":"%s","version":%d,"size":%d,"hash":"%s"}`, name, version, size, hash)

	client := http.Client{}
	url := fmt.Sprintf("http://%s/metadata/_doc/%s_%d", os.Getenv("ES_SERVER"), name, version)
	request, _ := http.NewRequest("POST", url, strings.NewReader(doc))
	request.Header.Set("Content-Type", "application/json")
	r, e := client.Do(request)
	if e != nil {
		return e
	}

	if r.StatusCode == http.StatusConflict {
		return PutMetadata(name, version+1, size, hash)
	}

	if r.StatusCode != http.StatusCreated {
		result, _ := ioutil.ReadAll(r.Body)
		return fmt.Errorf("fail to put metadata: %d %s", r.StatusCode, string(result))
	}

	return nil
}

// 添加元数据，版本加一
func AddVersion(name, hash string, size int64) error {
	version, e := SearchLatestVersion(name)
	if e != nil {
		return e
	}

	return PutMetadata(name, version.Version+1, size, hash)
}

// 获取所有元数据，指定 name 返回对应文件所以版本元信息
func SearchAllVersion(name string, from, size int) ([]Metadata, error) {
	url := fmt.Sprintf("http://%s/metadata/_search", os.Getenv("ES_SERVER"))
	doc := fmt.Sprintf(`{"query":?,"sort": [{"version":{"order":"desc"}}],"from":%d,"size":%d}`, from, size)

	if strings.Trim(name, " ") != "" {
		doc = strings.Replace(doc, "?", fmt.Sprintf(`{"match":{"name":"%s"}}`, name), 1)
	} else {
		doc = strings.Replace(doc, "?", `{"match_all":{}}`, 1)
		log.Println(doc)
	}

	client := http.Client{}
	request, _ := http.NewRequest("GET", url, strings.NewReader(doc))
	request.Header.Set("Content-Type", "application/json")
	r, e := client.Do(request)
	if e != nil {
		return nil, e
	}

	metas := make([]Metadata, 0)
	result, _ := ioutil.ReadAll(r.Body)
	var sr searchResult
	json.Unmarshal(result, &sr)

	for i := range sr.Hits.Hits {
		metas = append(metas, sr.Hits.Hits[i].Source)
	}

	return metas, nil
}
