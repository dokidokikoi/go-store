package rabbitmq

import (
	"encoding/json"

	"github.com/streadway/amqp"
)

// RabbitMQ结构图
type RabbitMQ struct {
	channel  *amqp.Channel
	Name     string
	exchange string
}

// 连接RabbitMQ服务，声明一个消息队列
func New(s string) *RabbitMQ {
	conn, e := amqp.Dial(s)
	if e != nil {
		panic(e)
	}

	ch, e := conn.Channel()
	if e != nil {
		panic(e)
	}

	q, e := ch.QueueDeclare(
		"",
		false,
		true,
		false,
		false,
		nil)
	if e != nil {
		panic(e)
	}

	mq := new(RabbitMQ)
	mq.channel = ch
	mq.Name = q.Name
	return mq
}

// 消息队列绑定交换机
func (q *RabbitMQ) Bind(exchange string) {
	e := q.channel.QueueBind(
		q.Name,
		"",
		exchange,
		false,
		nil)
	if e != nil {
		panic(e)
	}

	q.exchange = exchange
}

// 向消息队列发布消息
func (q *RabbitMQ) Send(queue string, body interface{}) {
	str, e := json.Marshal(body)
	if e != nil {
		panic(e)
	}

	e = q.channel.Publish(
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ReplyTo: q.Name,
			Body:    []byte(str),
		})
	if e != nil {
		panic(e)
	}
}

// 向交换机发送消息
func (q *RabbitMQ) Publish(excahnge string, body interface{}) {
	str, e := json.Marshal(body)
	if e != nil {
		panic(e)
	}

	e = q.channel.Publish(
		excahnge,
		"",
		false,
		false,
		amqp.Publishing{
			ReplyTo: q.Name,
			Body:    []byte(str),
		})
	if e != nil {
		panic(e)
	}
}

// 消费消息
func (q *RabbitMQ) Consume() <-chan amqp.Delivery {
	c, e := q.channel.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil)
	if e != nil {
		panic(e)
	}

	return c
}

// 关闭连接
func (q *RabbitMQ) Close() {
	q.channel.Close()
}
