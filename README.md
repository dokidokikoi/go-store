# go-store
golang 分布式对象存储

本项目来自，由胡世杰先生编著的《分布式对象存储--原理，架构及GO语言实现》一书

分八个版本，对应书中八个章节

```

├── 对象存储简介
└── v1
├── 可推展的分布式系统
└── v2
├── 元数据服务
└── v3
├── 数据校验与去重
└── v4
├── 数据冗余和即时修复
└── v5
├── 断点续传
└── v6
├── 数据压缩
└── v7
├── 数据维护
└── v8
```



项目涉及到的 rabbitmq 和 elaticsearch 的知识可以看我做的笔记



# v1 对象存储简介

```
.
├── go.mod
├── main.go
└── objects
    └── handler.go
```

第一个版本非常简单

支持两种请求，put 和 get，对应上传文件和下载文件

上传文件的流程是：服务端收到 http 请求，将请求体  body 中的文本写入到 url 指定名称的文件中

下载文件的流程是：服务端收到 http 请求，将 url 中指定名称的文件的内容写入到响应体中



启动命令：

```
# LISTEN_ADDRESS 监听端口
# STORE_ROOT 文件目录
LISTEN_ADDRESS=:8888 STORE_ROOT=/object/1 go run main.go
```

当然目录要提前建好



# v2 可推展的分布式系统

```
.
├── apiservice
│   ├── go.mod
│   ├── go.sum
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   ├── objects
│   │   └── handler.go
│   ├── objectstream
│   │   └── objectstream.go
│   └── rabbitmq
│       └── rabbitmq.go
├── dataservice
│   ├── go.mod
│   ├── go.sum
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   ├── objects
│   │   └── handler.go
│   └── rabbitmq
│       └── rabbitmq.go
└── script
    └── startup.sh
```

v1 只是最简单的单体服务，我们现在实现可扩展的分布式存储

将单体服务拆分为数据服务与接口服务。服务间的通信采用 rabbitmq，在 rabbitmq 服务器上新建两个交换机 apiServers 和 dataServers（类型都是fanout） 。

数据服务向 apiServers 发送心跳信息（服务地址），接口服务5秒未收到心跳信息，认为该数据服务出现异常。

接口服务向 dataServers 发送文件信息，新建一个临时消息队列，消息的正文是需要定位的对象,返回地址则是该临时队列的名字。如果某数据服务有该文件，向接口服务的临时队列发送自己的服务地址，接口服务收到后调用数据服务的下载文件接口（超时时间为1秒）



业务流程如下：

![image-20220906093819142](../../.config/Typora/typora-user-images/image-20220906093819142.png)

启动脚本：

```bash
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
```





# v3 元数据服务

```
.
├── apiservice
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   ├── objects
│   │   └── handler.go
│   ├── objectstream
│   │   └── objectstream.go
│   └── versions
│       └── veriosn.go
├── dataservice
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   └── objects
│       └── handler.go
├── es
│   ├── hit.go
│   └── meta.go
├── go.mod
├── go.sum
├── rabbitmq
│   └── rabbitmq.go
├── script
│   └── startup.sh
└── utils
    └── hash.go
```

除了那些系统定义的元数据以外,用户也可以为这个对象添加自定义的元数据

我们给文件定义的元数据是

```
name	名称
version	版本
size	大小
hash	散列值
```

对象的散列值是一种非常特殊的元数据,因为对象存储通常将对象的散列值作为其全局唯一的标识符。在此前,数据服务节点上的对象都是用名字来引用的,如果两个对象名字不同,那么我们无法知道它们的内容是否相同。这让我们无法实现针对不同对象的去重。

现在,以对象的散列值作为标识符,我们就可以将接口服务层访问的对象和数据服务存取的对象数据解耦合。客户端和接口服务通过对象的名字来引用一个对象,而实际则是通过其散列值来引用存储在数据节点上的对象数据,只要散列值相同则可以认为对象的数据相同,这样就可以实现名字不同但数据相同的对象之间的去重。



对象的散列值是通过散列函数计算出来的,散列函数会将对象的数据进行重复多轮的数学运算,这些运算操作包括按位与、按位或、按位异或等,最后计算出来一个长度固定的数字,作为对象的散列值。一个理想的散列函数具有以下5个特征：

* 操作具有决定性,同样的数据必定计算出同样的散列值。
* 无论计算任何数据都很快。
* 无法根据散列值倒推数据,只能遍历尝试所有可能的数据。
* 数据上微小的变化就会导致散列值的巨大改变,新散列值和旧散列值不具有相关性。
* 无法找到两个能产生相同散列值的不同数据。



可惜这只是理想的情况,现实世界里不可能完全满足。在现实世界,一个散列函数 hash 的安全级别根据以下 3 种属性决定：

* 抗原像攻击: 给定一个散列值 h ,难以找到一个数据 m 令 h=hash(m)。这个属性称为函数的单向性。欠缺单向性的散列函数易受到原像攻击。
* 抗第二原像攻击: 给定一个数据 m1,难以找到第二个数据 m2 令 hash(m1)= hash(m2) 。欠缺该属性的散列函数易受到第二原像攻击。
* 抗碰撞性: 难以找到两个不同的数据 m1 和 m2 令 hash(m1)=hash(m2)。这样的一对数据被称为散列碰撞。



本项目使用的散列函数是 SHA-256,该函数使用64轮的数学运算,产生一个长度为256位的二进制数字作为散列值



我们使用 es（ElasticSearch）保存元数据

> 目前ES的索引的主分片(primary shards)数量一旦被创建就无法更改,
>
> 对于对象存储来说这会导致元数据服务的容量无法自由扩容.
>
> 对于有扩展性需求的开发者，推荐的一种解决方法是使用ES滚动索引(rollover index)。使用滚动索引之后,只要当前索引中的文档数量超出设定的阈值,ES就会自动创建一个新的索引用于数据的插入,而数据的搜索则依然可以通过索引的别名访问之前所有的旧索引



es 新建索引：

```json
PUT /metadata
{
  "mappings": {
    "properties": {
      "name": {
        "type": "text",
        "index": "true"
      },
      "version": {
        "type": "integer"
      },
      "size": {
        "type": "integer"
      },
      "hash": {
        "type": "keyword"
      }
    }
  }
}
```



es 新建文档数据：

```json
POST /metadata/_doc/testone_1
{
  "name": "testone",
  "version": "1",
  "size": 1000,
  "hash": ""
}
```



业务逻辑：

![image-20220906111817229](https://harukaze-blog.oss-cn-shenzhen.aliyuncs.com/article/image-20220906111817229.png)

```bash
#! /bin/bash

for i in `seq 1 6`
do
    mkdir -p /tmp/$i/object
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
```



计算 hash：

```sh
echo -n "this is object test3" | openssl dgst -sha256 –binary | base64
```







# v4 数据校验与去重

```
.
├── apiservice
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   ├── objects
│   │   └── handler.go
│   ├── objectstream
│   │   └── objectstream.go
│   └── versions
│       └── veriosn.go
├── dataservice
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   ├── objects
│   │   └── handler.go
│   └── temp
│       └── temp.go
├── es
│   ├── hit.go
│   └── meta.go
├── go.mod
├── go.sum
├── rabbitmq
│   └── rabbitmq.go
├── script
│   └── startup.sh
└── utils
    └── hash.go
```

由于各种原因，从客户端传给服务的 hash 值与上传的数据可能不一致，因此我们必须进行数据校验,验证客户端提供的散列值和我们自己根据对象数据计算出来的散列值是否一致



但是这时就出现了一个大问题：

一直以来我们都是以数据流的形式处理来自客户端的请求,接口服务调用 `io.Copy` 从对象 PUT 请求的正文中直接读取对象数据并写入数据服务。这是因为客户端上传的对象大小可能超出接口服务节点的内存,我们不能把整个对象读入内存后再进行处理。而现在我们必须等整个对象都上传完以后才能算出散列值,然后才能决定是否要存进数据服务。

这就形成了一个悖论:在客户端的对象完全上传完毕之前,我们不知道要不要把这个对象写入数据服务;但是等客户端的对象上传完毕之后再开始写入我们又做不到,因为对象可能太大,内存里根本放不下。

并且，考虑到在后面还会对数据进行压缩，这就表示不能在数据服务节点进行数据校
验,将校验一致的对象保留,不一致的删除。因为压缩之后的数据显然和未压缩的数据完全不同。我们就无法在数据服务节点上进行数据校验。数据校验这一步骤必须在接口服务节点完成。



方案：

为了真正解决上述矛盾,我们需要在数据服务上提供对象的缓存功能,接口服务不需要将用户上传的对象缓存在自身节点的内存里,而是传输到某个数据服务节点的一个临时对象里,并在传输数据的同时计算其散列值。当整个数据传输完毕以后,散列值计算也同步完成,如果一致,接口节点需要将临时对象转成正式对象;如果不一致,则将临时对象删除



启动脚本：

```bash
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
```







# v5 数据冗余和即时修复

```
.
├── apiservice
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   ├── objects
│   │   └── handler.go
│   ├── objectstream
│   │   └── objectstream.go
│   ├── rs
│   │   ├── getStream.go
│   │   └── putStream.go
│   └── versions
│       └── veriosn.go
├── dataservice
│   ├── heartbeat
│   │   └── heartbeat.go
│   ├── locate
│   │   └── locate.go
│   ├── main.go
│   ├── objects
│   │   └── handler.go
│   └── temp
│       └── temp.go
├── es
│   ├── hit.go
│   └── meta.go
├── go.mod
├── go.sum
├── rabbitmq
│   └── rabbitmq.go
├── script
│   └── startup.sh
├── types
│   └── locateMessage.go
└── utils
    └── hash.go
```

