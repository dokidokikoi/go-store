package objectstream

import (
	"fmt"
	"io"
	"net/http"
)

type PutStream struct {
	writer *io.PipeWriter
	c      chan error
}

func NewPutStream(server, object string) *PutStream {
	reader, writer := io.Pipe()
	c := make(chan error)
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
