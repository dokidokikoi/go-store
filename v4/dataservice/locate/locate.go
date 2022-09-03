package locate

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"store/rabbitmq"
)

var objects = make(map[string]int)
var mutex sync.Mutex

func Locate(hash string) bool {
	mutex.Lock()
	_, ok := objects[hash]
	mutex.Unlock()
	return ok
}

func StartLocate() {
	q := rabbitmq.New(os.Getenv("RABBITMQ_SERVER"))
	defer q.Close()
	q.Bind("dataServers")
	c := q.Consume()

	for msg := range c {
		hash, e := strconv.Unquote(string(msg.Body))
		if e != nil {
			panic(e)
		}

		exist := Locate(hash)
		if exist {
			q.Send(msg.ReplyTo, os.Getenv("LISTEN_ADDRESS"))
		}
	}
}

func CollectObjects() {
	files, _ := filepath.Glob(os.Getenv("STORE_ROOT") + "/object/*")
	for i := range files {
		hash := filepath.Base(files[i])
		objects[hash] = 1
	}
}

func Add(hash string) {
	mutex.Lock()
	defer mutex.Unlock()
	objects[hash] = 1
}

func Del(hash string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(objects, hash)
}
