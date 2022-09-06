package heartbeat

import (
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"apiservice/rabbitmq"
)

// 数据服务，键是服务地址，值是最后心跳时间
var dataServers = make(map[string]time.Time)

// 对 dataServers 操作时需要加锁保证数据安全
var mutex sync.Mutex

// 监听数据服务心跳
func ListenHeartbeat() {
	q := rabbitmq.New(os.Getenv("RABBITMQ_SERVER"))
	defer q.Close()

	q.Bind("apiServers")
	c := q.Consume()

	go removeExpireDataServer()

	for msg := range c {
		dataServer, e := strconv.Unquote(string(msg.Body))
		if e != nil {
			panic(e)
		}

		mutex.Lock()
		dataServers[dataServer] = time.Now()
		mutex.Unlock()
	}
}

// 移除过期的数据服务
func removeExpireDataServer() {
	for {
		time.Sleep(5 * time.Second)
		mutex.Lock()
		for s, t := range dataServers {
			if t.Add(10 * time.Second).Before(time.Now()) {
				delete(dataServers, s)
			}
		}
		mutex.Unlock()
	}
}

func GetDataServers() []string {
	mutex.Lock()
	defer mutex.Unlock()
	ds := make([]string, 0)
	for s := range dataServers {
		ds = append(ds, s)
	}

	return ds
}

// 返回随机数据服务地址
func ChooseRandomDataServer() string {
	ds := GetDataServers()
	n := len(ds)
	if n == 0 {
		return ""
	}

	return ds[rand.Intn(n)]
}
