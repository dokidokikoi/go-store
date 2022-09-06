package locate

import (
	"os"
	"strconv"

	"dataservice/rabbitmq"
)

// 本地存储是否存在该文件
func Locate(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

// 监听交换机发送的消息
func StartLocate() {
	q := rabbitmq.New(os.Getenv("RABBITMQ_SERVER"))
	defer q.Close()
	q.Bind("dataServers")
	c := q.Consume()

	for msg := range c {
		object, e := strconv.Unquote(string(msg.Body))
		if e != nil {
			panic(e)
		}
		if Locate(os.Getenv("STORE_ROOT") + "/object/" + object) {
			q.Send(msg.ReplyTo, os.Getenv("LISTEN_ADDRESS"))
		}
	}
}
