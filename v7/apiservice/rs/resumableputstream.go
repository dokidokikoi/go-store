package rs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"store/apiservice/objectstream"
	"store/utils"
)

type resumableToken struct {
	Name    string
	Size    int64
	Hash    string
	Servers []string
	Uuids   []string
}

type RsResumablePutStream struct {
	*RSPutStream
	*resumableToken
}

func NewRSResumablePutGetStream(dataServers []string, name, hash string, size int64) (*RsResumablePutStream, error) {
	putStream, e := NewRSPutStream(dataServers, hash, size)
	if e != nil {
		return nil, e
	}

	uuids := make([]string, ALL_SHARDS)
	for i := range uuids {
		uuids[i] = putStream.writers[i].(*objectstream.TempPutStream).Uuid
	}

	token := &resumableToken{
		Name:    name,
		Size:    size,
		Hash:    hash,
		Servers: dataServers,
		Uuids:   uuids,
	}

	return &RsResumablePutStream{putStream, token}, nil
}

func (s *RsResumablePutStream) ToToken() string {
	b, _ := json.Marshal(s)

	return base64.StdEncoding.EncodeToString(b)
}

// 将自身数据以JSON格式编入,然后返回经过Base64编码后的字符串
func NewRSResumablePutStreamFromToken(token string) (*RsResumablePutStream, error) {
	b, e := base64.StdEncoding.DecodeString(token)
	if e != nil {
		return nil, e
	}

	var t resumableToken
	e = json.Unmarshal(b, &t)
	if e != nil {
		return nil, e
	}

	writers := make([]io.Writer, ALL_SHARDS)
	for i := range writers {
		writers[i] = &objectstream.TempPutStream{
			Server: t.Servers[i],
			Uuid:   t.Uuids[i],
		}
	}

	enc := newEncoder(writers)

	return &RsResumablePutStream{&RSPutStream{enc}, &t}, nil
}

// 获取已上传文件大小
func (s *RsResumablePutStream) CurrentSize() int64 {
	r, e := http.Head(fmt.Sprintf("http://%s/temp/%s", s.Servers[0], s.Uuids[0]))
	if e != nil {
		log.Println(e)

		return -1
	}
	if r.StatusCode != http.StatusOK {
		log.Println(r.StatusCode)

		return -1
	}

	size := utils.GetSizeFromHeader(r.Header) * DATA_SHARDS
	if size > s.Size {
		size = s.Size
	}

	return size
}

type RSResumableGetStream struct {
	*RSGetStream
}

// 获取临时文件输入流
func NewRSResumableGetStream(dataServers []string, uuids []string, size int64) (*RSResumableGetStream, error) {
	readers := make([]io.Reader, ALL_SHARDS)
	for i := range dataServers {
		resp, e := http.Get(fmt.Sprintf("http://%s/temp/%s", dataServers[i], uuids[i]))
		if e != nil {
			log.Panicln(e)
			return nil, e
		}

		readers[i] = resp.Body
	}

	doc := NewDecoder(readers, nil, size)

	return &RSResumableGetStream{&RSGetStream{doc}}, nil
}
