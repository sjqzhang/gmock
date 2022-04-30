package main

import (
	"context"
	"fmt"
	"github.com/sjqzhang/gmock"
	"io/ioutil"
	"net/http"
	"time"
)

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func IntToPrt(i int) *int {
	v := i
	return &v
}
func StrToPrt(s string) *string {
	v := s
	return &v
}

func main() {
	testMockDB()
	testMockRedis()
	testMockHttpServer()
}

func testMockDB() {
	mockdb := gmock.NewMockDB("mock.sql")
	mockdb.RegisterModels(&User{})
	mockdb.Reset()
	user := User{
		Id:   1,
		Name: "hello",
		Age:  100,
	}
	var user2 User
	db := mockdb.GetGormDB()
	//db.AutoMigrate(&User{})
	err := db.Create(&user).Error
	if err != nil {
		panic(err)
	}
	db.Where("id=?", 1).Find(&user2)
	if user.Name != user2.Name {
		panic("mock db panic")
	}

}

func testMockRedis() {
	server := gmock.NewMockRedisServer()
	client := server.GetRedisClient()
	ctx := context.Background()
	key := "aa"
	value := "aa value"
	client.Set(ctx, key, value, time.Second*10)
	cmd := client.Get(ctx, key)
	if cmd.Val() != value {
		panic("redis")
	}

}

func testMockHttpServer() {
	server := gmock.NewMockHttpServer("./", []string{"www.baidu.com"})
	server.InitMockHttpServer()
	resp, err := http.Get("http://www.baidu.com/testRequest")
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

}
