package objectstream

import (
	"fmt"
	"io"
	"net/http"
)

// 实现 io.Writer
type PutStream struct {
	writer *io.PipeWriter
	// c用于把在一个goroutine传输数据的过程中发生的错误传回主线程。
	c chan error
}

func NewPutStream(server, object string) *PutStream {
	// reader, writer 是管道互联的,写入writer的内容可以从reader中读出来。
	// 能以写入数据流的方式操作HTTP的PUT请求
	reader, writer := io.Pipe()
	c := make(chan error)

	// 由于管道的读写是阻塞的,我们需要在一个goroutine中调用client.Do方法
	go func() {
		request, _ := http.NewRequest("PUT", "http://"+server+"/object/"+object, reader)
		client := http.Client{}
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

// PutStream.Close 方法用于关闭 writer。
// 这是为了让管道另一端的 reader 读到 io.EOF,否则在 goroutine 中运行的 client.Do 将始终阻塞无法返回
func (w *PutStream) Close() error {
	w.writer.Close()

	return <-w.c
}

// 实现 io.Reader
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
