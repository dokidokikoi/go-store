package locate

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"store/rabbitmq"
)

// 存储目录下的对象文件
// 避免每次查询文件是否存在时访问磁盘，将所有文件的散列值读入内存
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
	// 读取存储目录里的所有文件
	files, _ := filepath.Glob(os.Getenv("STORE_ROOT") + "/object/*")

	// 获取每个文件的文件名（散列值），加入缓存
	for i := range files {
		hash := filepath.Base(files[i])
		objects[hash] = 1
	}
}

// 将用户上传的文件加入内存
func Add(hash string) {
	mutex.Lock()
	defer mutex.Unlock()
	objects[hash] = 1
}

// 将文件移出内存
func Del(hash string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(objects, hash)
}
