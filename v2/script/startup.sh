#! /bin/bash

for i in `seq 1 6`
do
    mkdir -p /tmp/$i/object
done

export RABBITMQ_SERVER=amqp://harukaze:123456@localhost:5672

cd ../dataservice/

echo $PWD

LISTEN_ADDRESS=localhost:12345 STORE_ROOT=/tmp/1 go run main.go &
LISTEN_ADDRESS=localhost:12346 STORE_ROOT=/tmp/2 go run main.go &
LISTEN_ADDRESS=localhost:12347 STORE_ROOT=/tmp/3 go run main.go &
LISTEN_ADDRESS=localhost:12348 STORE_ROOT=/tmp/4 go run main.go &
LISTEN_ADDRESS=localhost:12349 STORE_ROOT=/tmp/5 go run main.go &
LISTEN_ADDRESS=localhost:12350 STORE_ROOT=/tmp/6 go run main.go &

cd ../apiservice/

LISTEN_ADDRESS=localhost:12351 go run main.go &
LISTEN_ADDRESS=localhost:12352 go run main.go
