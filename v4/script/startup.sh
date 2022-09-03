#! /bin/bash

for i in `seq 1 6`
do
    mkdir -p /tmp/$i/object
    mkdir /tmp/$i/temp
done

export RABBITMQ_SERVER=amqp://harukaze:123456@localhost:5672
export ES_SERVER=localhost:9200

LISTEN_ADDRESS=localhost:12345 STORE_ROOT=/tmp/1 go run dataservice/main.go &
LISTEN_ADDRESS=localhost:12346 STORE_ROOT=/tmp/2 go run dataservice/main.go &
LISTEN_ADDRESS=localhost:12347 STORE_ROOT=/tmp/3 go run dataservice/main.go &
LISTEN_ADDRESS=localhost:12348 STORE_ROOT=/tmp/4 go run dataservice/main.go &
LISTEN_ADDRESS=localhost:12349 STORE_ROOT=/tmp/5 go run dataservice/main.go &
LISTEN_ADDRESS=localhost:12350 STORE_ROOT=/tmp/6 go run dataservice/main.go &

LISTEN_ADDRESS=localhost:12351 go run apiservice/main.go &
LISTEN_ADDRESS=localhost:12352 go run apiservice/main.go
