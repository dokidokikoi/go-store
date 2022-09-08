package locate

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"store/rabbitmq"
	"store/types"
)

var objects = make(map[string]int)
var mutex sync.Mutex

func Locate(hash string) int {
	mutex.Lock()
	id, ok := objects[hash]
	mutex.Unlock()

	if !ok {
		return -1
	}

	return id
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

		id := Locate(hash)
		if id != -1 {
			q.Send(msg.ReplyTo, types.LocateMessage{
				Addr: os.Getenv("LISTEN_ADDRESS"),
				Id:   id,
			})
		}
	}
}

func CollectObjects() {
	files, _ := filepath.Glob(os.Getenv("STORE_ROOT") + "/object/*")
	for i := range files {
		file := strings.Split(filepath.Base(files[i]), ".")
		if len(file) != 3 {
			panic(files[i])
		}
		hash := file[0]
		id, e := strconv.Atoi(file[1])
		if e != nil {
			panic(e)
		}

		objects[hash] = id
	}
}

func Add(hash string, id int) {
	mutex.Lock()
	defer mutex.Unlock()
	objects[hash] = id
}

func Del(hash string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(objects, hash)
}
