package rs

import (
	"fmt"
	"io"

	"store/apiservice/objectstream"

	"github.com/klauspost/reedsolomon"
)

const (
	// 数据片个数
	DATA_SHARDS = 4
	// 校验片个数
	PARITY_SHARDS = 2
	// 总个数
	ALL_SHARDS = DATA_SHARDS + PARITY_SHARDS
	// 块大小
	BLOCK_PER_SHARD = 8000
	// 缓存大小
	BLOCK_SIZE = BLOCK_PER_SHARD * DATA_SHARDS
)

type encoder struct {
	writers []io.Writer
	// reedsolomon包是一个RS编解码的开源库
	enc   reedsolomon.Encoder
	cache []byte
}

// ncoder 的 Write 方法在 for 循环里将 p 中待写入的数据以块的形式放入缓存,
// 如果缓存已满就调用 Flush 方法将缓存实际写入 writers。
// 缓存的上限是每个数据片 BLOCK_PER_SHARD 字节, DATA_SHARDS 个数据片共 BLOCK_SIZE  字节。
// 如果缓存里剩余的数据不满 BLOCK_SIZE 字节就暂不刷新,等待 Write 方法下一次被调用。
func (e *encoder) Write(p []byte) (n int, err error) {
	length := len(p)
	current := 0

	for length != 0 {
		next := BLOCK_SIZE - len(e.cache)

		// next > length 说明数据已经到最后了
		if next > length {
			next = length
		}
		e.cache = append(e.cache, p[current:current+next]...)
		if len(e.cache) == BLOCK_SIZE {
			e.Flush()
		}
		current += next
		length -= next
	}

	return len(p), nil
}

func (e *encoder) Flush() {
	if len(e.cache) == 0 {
		return
	}

	shards, _ := e.enc.Split(e.cache)
	e.enc.Encode(shards)
	for i := range shards {
		e.writers[i].Write(shards[i])
	}

	e.cache = []byte{}
}

func newEncoder(writers []io.Writer) *encoder {
	enc, _ := reedsolomon.New(DATA_SHARDS, PARITY_SHARDS)

	return &encoder{
		writers: writers,
		enc:     enc,
		cache:   nil,
	}
}

type RSPutStream struct {
	*encoder
}

func NewRSPutStream(dataServers []string, hash string, size int64) (*RSPutStream, error) {
	if len(dataServers) != ALL_SHARDS {
		return nil, fmt.Errorf("dataServers number mismatch")
	}

	// 每个分片的大小,size/DATA_SHARDS 再向上取整
	perShard := (size + DATA_SHARDS - 1) / DATA_SHARDS
	writers := make([]io.Writer, ALL_SHARDS)
	var e error

	for i := range writers {
		writers[i], e = objectstream.NewTempPutStream(
			dataServers[i],
			fmt.Sprintf("%s.%d", hash, i),
			perShard)
		if e != nil {
			return nil, e
		}
	}
	enc := newEncoder(writers)

	return &RSPutStream{enc}, nil
}

func (s *RSPutStream) Commit(flag bool) {
	// 将剩下的缓存写入
	s.Flush()
	for i := range s.writers {
		s.writers[i].(*objectstream.TempPutStream).Commit(flag)
	}
}
