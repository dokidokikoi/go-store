package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func GetHashFromHeader(h http.Header) string {
	digest := h.Get("digest")
	if len(digest) < 9 {
		return ""
	}

	if digest[:8] != "SHA-256=" {
		return ""
	}

	return digest[8:]
}

func GetSizeFromHeader(h http.Header) int64 {
	size, _ := strconv.ParseInt(h.Get("content-length"), 0, 64)

	return size
}

func SetHash(hash string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(hash, "/", ","), "=", "."), "+", "~")
}

func GetHash(hash string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(hash, ",", "/"), ".", "="), "~", "+")
}

// 计算数据散列值并返回
func CalculateHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		log.Println(err)
		return "", err
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
