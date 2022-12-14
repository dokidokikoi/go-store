package heartbeat

import (
	"os"
	"time"

	"dataservice/rabbitmq"
)

// 每隔 5s 发送心跳
func StartHeartbeat() {
	q := rabbitmq.New(os.Getenv("RABBITMQ_SERVER"))
	defer q.Close()

	for {
		q.Publish("apiServers", os.Getenv("LISTEN_ADDRESS"))
		time.Sleep(5 * time.Second)
	}
}
