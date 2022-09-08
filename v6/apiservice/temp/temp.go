package temp

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"store/apiservice/locate"
	"store/apiservice/rs"
	"store/es"
	"store/utils"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m == http.MethodHead {
		head(w, r)
		return
	}
	if m == http.MethodPut {
		put(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// 上传文件
func put(w http.ResponseWriter, r *http.Request) {
	token := strings.Split(r.URL.EscapedPath(), "/")[2]
	stream, e := rs.NewRSResumablePutStreamFromToken(utils.GetHash(token))

	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusForbidden)

		return
	}

	// 已经上传的文件大小
	current := stream.CurrentSize()
	if current == -1 {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	offset := utils.GetOffsetFromHeader(r.Header)
	if current != offset {
		// 续传的文件偏移量与已上传文件大小不匹配
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)

		return
	}

	// 缓冲
	bytes := make([]byte, rs.BLOCK_SIZE)
	for {
		n, e := io.ReadFull(r.Body, bytes)
		if e != nil && e != io.EOF && e != io.ErrUnexpectedEOF {
			log.Println(e)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		// 上传文件大于文件实际大小
		current += int64(n)
		if current > stream.Size {
			stream.Commit(false)
			log.Println("resumable put exceed size")
			w.WriteHeader(http.StatusForbidden)

			return
		}

		// 如果某次读取的长度不到 BLOCK_SIZE 字节且读到的总长度不等于对象的大小,
		// 说明本次客户端上传结束,还有后续数据需要上传。
		// 此时接口服务会丢弃最后那次读取的长度不到32 000字节的数据

		// 将这部分数据缓存在接口服务的内存里没有意义,
		// 下次客户端不一定还访问同一个接口服务节点。
		// 而如果我们将这部分数据直接写入临时对象,
		// 那么我们就破坏了每个数据片以8000字节为一个块写入的约定,在读取时就会发生错误
		if n != rs.BLOCK_SIZE && current != stream.Size {
			return
		}

		stream.Write(bytes[:n])

		// 文件写完了
		if current == stream.Size {
			// 将最后的缓冲内的数据写入
			stream.Flush()

			getStream, e := rs.NewRSResumableGetStream(stream.Servers, stream.Uuids, stream.Size)
			if e != nil {
				log.Println(e)
				w.WriteHeader(http.StatusInternalServerError)
			}

			hash, _ := utils.CalculateHash(getStream)

			log.Println(hash, stream.Hash)

			if utils.SetHash(hash) != stream.Hash {
				stream.Commit(false)
				log.Println("resumable put done but hash mismatch")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			if locate.Exist(utils.SetHash(hash)) {
				stream.Commit(false)
			} else {
				stream.Commit(true)
			}

			e = es.AddVersion(stream.Name, stream.Hash, stream.Size)
			if e != nil {
				log.Println(e)
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}
	}
}

// 获取已经上传文件大小
func head(w http.ResponseWriter, r *http.Request) {
	token := strings.Split(r.URL.EscapedPath(), "/")[2]
	stream, e := rs.NewRSResumablePutStreamFromToken(utils.GetHash(token))
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusForbidden)

		return
	}

	current := stream.CurrentSize()
	if current == -1 {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	w.Header().Set("content-length", fmt.Sprintf("%d", current))
}
