package rs

import (
	"fmt"
	"io"

	"store/apiservice/objectstream"

	"github.com/klauspost/reedsolomon"
)

type RSGetStream struct {
	*decoder
}

func NewRSGetStream(locateinfo map[int]string, dataServers []string, hash string, size int64) (*RSGetStream, error) {
	if len(locateinfo)+len(dataServers) != ALL_SHARDS {
		return nil, fmt.Errorf("dataServers number mismatch")
	}

	readers := make([]io.Reader, ALL_SHARDS)
	for i := 0; i < ALL_SHARDS; i++ {
		server := locateinfo[i]
		if server == "" {
			locateinfo[i] = dataServers[0]
			dataServers = dataServers[1:]
			continue
		}

		reader, e := objectstream.NewGetStream(server, fmt.Sprintf("%s.%d", hash, i))
		if e == nil {
			readers[i] = reader
		}
	}

	writers := make([]io.Writer, ALL_SHARDS)
	perShard := (size + DATA_SHARDS - 1) / DATA_SHARDS
	var e error

	for i := range readers {
		// 如果 reader 为空则创建临时对象写入流用于恢复分片
		if readers[i] == nil {
			writers[i], e = objectstream.NewTempPutStream(locateinfo[i], fmt.Sprintf("%s.%d", hash, i), perShard)
			if e != nil {
				return nil, e
			}
		}
	}

	doc := NewDecoder(readers, writers, size)

	return &RSGetStream{doc}, nil
}

// Close 时写入恢复分片
func (s *RSGetStream) Close() {
	for i := range s.writers {
		if s.writers[i] != nil {
			s.writers[i].(*objectstream.TempPutStream).Commit(true)
		}
	}
}

type decoder struct {
	readers []io.Reader
	writers []io.Writer
	// rs 解码
	enc reedsolomon.Encoder
	// 对象大小
	size int64
	// 用于缓存读取的数据
	cache     []byte
	cacheSize int
	// 当前已经读取了多少字节
	total int64
}

func NewDecoder(readers []io.Reader, writers []io.Writer, size int64) *decoder {
	enc, _ := reedsolomon.New(DATA_SHARDS, PARITY_SHARDS)

	return &decoder{
		readers:   readers,
		writers:   writers,
		enc:       enc,
		size:      size,
		cache:     nil,
		cacheSize: 0,
		total:     0,
	}
}

// 读缓存
func (d *decoder) Read(p []byte) (n int, err error) {
	if d.cacheSize == 0 {
		e := d.getData()
		if e != nil {
			return 0, e
		}
	}

	length := len(p)
	if d.cacheSize < length {
		length = d.cacheSize
	}

	d.cacheSize -= length
	copy(p, d.cache[:length])
	d.cache = d.cache[length:]

	return length, nil
}

// 写入缓存
func (d *decoder) getData() error {
	if d.total == d.size {
		return io.EOF
	}

	shards := make([][]byte, ALL_SHARDS)
	repairIds := make([]int, 0)
	for i := range shards {
		if d.readers[i] == nil {
			repairIds = append(repairIds, i)
		} else {
			shards[i] = make([]byte, BLOCK_PER_SHARD)
			n, e := io.ReadFull(d.readers[i], shards[i])
			if e != nil && e != io.EOF && e != io.ErrUnexpectedEOF {
				shards[i] = nil
			} else if n != BLOCK_PER_SHARD {
				shards[i] = shards[i][:n]
			}
		}
	}

	// 生成 shard 中为 nil 的分片
	e := d.enc.Reconstruct(shards)
	if e != nil {
		return e
	}

	// 修复缺失的分片
	for i := range repairIds {
		id := repairIds[i]
		d.writers[id].Write(shards[id])
	}

	// 遍历 DATA_SHARDS 个数据分片,将每个分片中的数据添加到缓存cache中,
	// 修改缓存当前的大小cacheSize以及当前已经读取的全部数据的大小total。
	for i := 0; i < DATA_SHARDS; i++ {
		shardSize := int64(len(shards[i]))

		// 防止填充数据被当成原始对象数据返回
		if d.total+shardSize > d.size {
			shardSize -= d.total + shardSize - d.size
		}

		d.cache = append(d.cache, shards[i][:shardSize]...)
		d.cacheSize += int(shardSize)
		d.total += shardSize
	}

	return nil
}
