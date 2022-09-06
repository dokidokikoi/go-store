package objectstream

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type PutStream struct {
	writer *io.PipeWriter
	c      chan error
}

type TempPutStream struct {
	Server string
	Uuid   string
}

func NewPutStream(server, object string) *PutStream {
	reader, writer := io.Pipe()
	c := make(chan error)
	go func() {
		request, _ := http.NewRequest("PUT", "http://"+server+"/object/"+object, reader)
		client := http.Client{}
		log.Println("http://" + server + "/object/" + object)
		r, e := client.Do(request)
		if e != nil && r.StatusCode != http.StatusOK {
			e = fmt.Errorf("dataServer return http code %d", r.StatusCode)
		}
		c <- e
	}()

	return &PutStream{writer, c}
}

func (w *PutStream) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

func (w *PutStream) Close() error {
	w.writer.Close()

	return <-w.c
}

type GetStream struct {
	reader io.Reader
}

func newGetStream(url string) (*GetStream, error) {
	r, e := http.Get(url)
	if e != nil {
		return nil, e
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dataServer return http code %d", r.StatusCode)
	}

	return &GetStream{
		r.Body,
	}, nil
}

func NewGetStream(server, object string) (*GetStream, error) {
	if server == "" || object == "" {
		return nil, fmt.Errorf("invalid server %s object %s", server, object)
	}

	return newGetStream("http://" + server + "/object/" + object)
}

func (r *GetStream) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func NewTempPutStream(server, hash string, size int64) (*TempPutStream, error) {
	// 将文件信息存到临时目录下文件名为生成的uuid
	request, e := http.NewRequest("POST", "http://"+server+"/temp/"+hash, nil)
	if e != nil {
		return nil, e
	}

	request.Header.Set("size", fmt.Sprintf("%d", size))
	client := http.Client{}
	response, e := client.Do(request)
	if e != nil {
		return nil, e
	}

	// 根据uuid将临时文件转正和删除
	uuid, e := ioutil.ReadAll(response.Body)
	if e != nil {
		return nil, e
	}

	return &TempPutStream{
		Server: server,
		Uuid:   string(uuid),
	}, nil
}

// 计算散列值时会被调用，将用户上传的数据保存到临时目录，等待校验结果
func (w *TempPutStream) Write(p []byte) (n int, err error) {
	request, e := http.NewRequest("PATCH", "http://"+w.Server+"/temp/"+w.Uuid, strings.NewReader(string(p)))
	if e != nil {
		return 0, e
	}

	client := http.Client{}
	r, e := client.Do(request)
	if e != nil {
		return 0, e
	}
	if r.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("dataserver return http code %d", r.StatusCode)
	}

	return len(p), nil
}

// 转正或删除临时文件
func (w *TempPutStream) Commit(flag bool) {
	method := "DELETE"
	if flag {
		method = "PUT"
	}

	request, _ := http.NewRequest(method, "http://"+w.Server+"/temp/"+w.Uuid, nil)
	client := http.Client{}
	client.Do(request)
}
