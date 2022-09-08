package locate

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"store/apiservice/rs"
	"store/rabbitmq"
	"store/types"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	info := Locate(strings.Split(r.URL.EscapedPath(), "/")[2])
	if len(info) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	b, _ := json.Marshal(info)
	w.Write(b)
}

func Locate(name string) (localinfo map[int]string) {
	q := rabbitmq.New(os.Getenv("RABBITMQ_SERVER"))
	q.Publish("dataServers", name)
	c := q.Consume()

	// 一秒后关闭临时消息队列
	go func() {
		time.Sleep(time.Second)
		q.Close()
	}()

	localinfo = make(map[int]string)

	// 接收 rs.ALL_SHARDS 个 types.LocateMessage
	for i := 0; i < rs.ALL_SHARDS; i++ {
		msg := <-c
		if len(msg.Body) == 0 {
			return
		}
		var info types.LocateMessage
		json.Unmarshal(msg.Body, &info)
		localinfo[info.Id] = info.Addr
	}

	return
}

func Exist(name string) bool {
	// 如果收到的数据小于必须的数量返回 false
	return len(Locate(name)) >= rs.DATA_SHARDS
}
