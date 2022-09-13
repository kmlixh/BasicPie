package BasicPie

import (
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"os"
)

func MD5(v string) string {
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil))
}
func GetPrivateKey(path string) (*rsa.PrivateKey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileInfo, _ := f.Stat()
	b := make([]byte, fileInfo.Size())
	f.Read(b)
	block, _ := pem.Decode(b)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes) //还原数据
	return privateKey, err
}
func UnMarshalJson(c *gin.Context, i interface{}) {
	bbs, er := io.ReadAll(c.Request.Body)
	if er != nil {
		panic(er)
	}
	fmt.Println(string(bbs))
	json.Unmarshal(bbs, i)
}
func GetPublicKey(path string) (*rsa.PublicKey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fileInfo, _ := f.Stat()
	b := make([]byte, fileInfo.Size())
	f.Read(b)
	block, _ := pem.Decode(b)
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes) //还原数据
	publicKey := pubKey.(*rsa.PublicKey)
	return publicKey, err
}
