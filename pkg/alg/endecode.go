package alg

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	log2 "log"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/golang/snappy"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/protocol/proto"
)

const (
	TcpHeadSize   = 2
	PacketMaxLen  = 512000 // 最大应用层包长度
	SnappySize    = 1 << 10
	decryptedData = "decrypted_data"
)

var (
	singKey = []byte("0b2a18e45d7df321")
)

type GameMsg struct {
	*proto.PacketHead
	Body pb.Message
}

func HandleFlag(flag uint32, body []byte) []byte {
	switch flag {
	case 0:
		// 不处理
		return body
	case 1:
		var dst []byte
		dst, _ = snappy.Decode(nil, body)
		return dst
	default:
		log2.Printf("Unknown flag:%d\n", flag)
		return body
	}
}

func UnGzip(bin []byte) ([]byte, error) {
	z, err := gzip.NewReader(bytes.NewReader(bin))
	if err != nil {
		return nil, err
	}
	defer z.Close()
	return io.ReadAll(z)
}

func CompGzip(bin []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	defer gz.Close()
	if _, err := gz.Write(bin); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func AESECB128Decode(key, ciphertext []byte) ([]byte, error) {
	if key == nil || ciphertext == nil {
		return nil, nil
	}
	key16 := make([]byte, 16)
	copy(key16, key)
	block, err := aes.NewCipher(key16)
	if err != nil {
		return nil, err
	}
	decrypted := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(decrypted[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}
	return PKCS7Unpadding(decrypted), nil
}

func PKCS7Unpadding(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	padding := int(data[length-1])
	if padding < 1 || padding > aes.BlockSize {
		return data
	}
	for i := 0; i < padding; i++ {
		if int(data[length-1-i]) != padding {
			return data
		}
	}
	return data[:length-padding]
}

func AESECB128Encode(key, plaintext []byte) ([]byte, error) {
	if key == nil || plaintext == nil {
		return nil, nil
	}
	key16 := make([]byte, 16)
	copy(key16, key)
	block, err := aes.NewCipher(key16)
	if err != nil {
		return nil, err
	}
	paddedText := PKCS7Padding(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(paddedText))
	for i := 0; i < len(paddedText); i += aes.BlockSize {
		block.Encrypt(ciphertext[i:i+aes.BlockSize], paddedText[i:i+aes.BlockSize])
	}

	return ciphertext, nil
}

func PKCS7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

type AutoReq struct {
	Data string `form:"data" binding:"required"`
	Sign string `form:"sign" binding:"required"`
	// ProductCode string `form:"productCode" binding:"required"`
}

type ResponseWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	context *gin.Context
}

func AutoCryptoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(AutoReq)
		if err := c.ShouldBind(req); err != nil {
			c.Abort()
			return
		}
		// 解密
		reqCiphertext, err := base64.StdEncoding.DecodeString(req.Data)
		if err != nil {
			c.Abort()
			return
		}
		reqPlainText, err := AESECB128Decode(singKey, reqCiphertext)
		if err != nil {
			c.Abort()
			return
		}
		// 签名
		if req.Sign != SingBytes(reqPlainText, singKey) {
			c.Abort()
			return
		}
		// debug
		// log.App.Debugf("auto :%s req:%s", c.Request.URL.Path, string(reqPlainText))
		// 写入请求
		c.Set(decryptedData, reqPlainText)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(reqPlainText))
		c.Request.ContentLength = int64(len(reqPlainText))
		c.Next()
	}
}

func DecryptedData(c *gin.Context, d any) error {
	data, exists := c.Get(decryptedData)
	if !exists {
		return errors.New("AutoCryptoMiddleware Err")
	}
	if err := sonic.Unmarshal(data.([]byte), d); err != nil {
		return err
	}
	return nil
}

func SingBytes(data, str []byte) string {
	mParams := make(map[string]interface{})
	err := sonic.Unmarshal(data, &mParams)
	if err != nil {
		return ""
	}
	return SignData(mParams, string(str))
}

func SignData(params map[string]interface{}, str string) string {
	if params == nil || len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buffer strings.Builder
	for _, key := range keys {
		buffer.WriteString(key)
		buffer.WriteString("=")
		buffer.WriteString(fmt.Sprintf("%v", params[key]))
		buffer.WriteString("&")
	}
	buffer.WriteString(str)

	return GetMD5Str(buffer.String())
}

func GetMD5Str(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	hashBytes := hash.Sum(nil)
	var hexBuilder strings.Builder
	for _, b := range hashBytes {
		hexStr := fmt.Sprintf("%02x", b&0xff)
		hexBuilder.WriteString(hexStr)
	}
	return hexBuilder.String()
}

func RandStr(length int, id uint32) string {
	key := make([]byte, length)
	rand.Read(key)
	return base64.URLEncoding.EncodeToString(key)
}
