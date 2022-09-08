package heartbeat

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"store/rabbitmq"
)

var dataServers = make(map[string]time.Time)
var mutex sync.Mutex

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

// 返回 n 个随机数据服务， exclude 是需要被排除的数据服务
// 因为当我们的定位完成后,实际收到的反馈消息有可能不足6个,
// 此时我们需要进行数据修复,根据目前已有的分片将丢失的分片复原出来并再次上传到数据服务
func ChooseRandomDataServers(n int, exclude map[int]string) (ds []string) {
	candidates := make([]string, 0)
	reverseExcludeMap := make(map[string]int)

	for id, addr := range exclude {
		reverseExcludeMap[addr] = id
	}

	servers := GetDataServers()

	for i := range servers {
		s := servers[i]
		_, excluded := reverseExcludeMap[s]
		if !excluded {
			candidates = append(candidates, s)
		}
	}

	length := len(candidates)
	log.Println(length, candidates)
	if length == n {
		return candidates
	}

	p := rand.Perm(length)

	for i := 0; i < n; i++ {
		ds = append(ds, candidates[p[i]])
		log.Println(i)
	}

	return
}
