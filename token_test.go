package basicPie

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

var client *redis.Client

func init() {
	client = redis.NewClient(&redis.Options{
		Addr: "10.0.1.5:6379",
		DB:   10, // use default DB
	})
	SetStore(NewRedisStore(client))
}

func TestGenToken(t *testing.T) {
	token, er := GenTokenForUser("1", "ADMIN", time.Hour*24)
	fmt.Println(token, er)
	if er != nil {
		t.Fatal(er)
	}
	result, er := store.GetTokenDetail(token)
	if result.Token != token {
		t.Fatal("can not find token")
	}
}
