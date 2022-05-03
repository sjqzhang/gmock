package main

import (
	"context"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/sjqzhang/gmock"
	"github.com/sjqzhang/gmock/mockdocker"
	"io/ioutil"
	"net/http"
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

	//testMockDB()
	//testMockDBV2()
	//testMockXorm()
	//testMockRedis()
	//testMockHttpServer()

	testMockDocker()

}

func testMockDB() {
	mockdb := gmock.NewMockDB("example")
	mockdb.RegisterModels(&User{})
	mockdb.ResetAndInit()
	db := mockdb.GetGormDB()
	var user User
	err := db.Where("id=?", 1).Find(&user).Error
	if err != nil {
		panic(err)
	}
	fmt.Println(user)

}
func testMockDBV2() {
	mockdb := gmock.NewMockDBV2("example")
	mockdb.RegisterModels(&User{})
	mockdb.ResetAndInit()
	db := mockdb.GetGormDB()
	var user User
	err := db.Where("id=?", 1).Find(&user).Error
	if err != nil {
		panic(err)
	}
	fmt.Println(user)

}

func testMockRedis() {
	server := gmock.NewMockRedisServer()
	client := server.GetRedisClient()
	ctx := context.Background()
	key := "aa"
	value := "aa value"
	pool := server.GetRedigoPool()
	conn := pool.Get()
	defer conn.Close()
	rep, err := conn.Do("set", key, value)
	if err != nil {
		panic(err)
	}
	fmt.Println(rep)
	//client.Set(ctx, key, value, time.Second*10)
	cmd := client.Get(ctx, key)
	if cmd.Val() != value {
		panic("redis")
	}

}

func testMockHttpServer() {
	server := gmock.NewMockHttpServer("./", []string{"www.baidu.com", "www.jenkins.org"})
	server.InitMockHttpServer()
	//server.SetReqRspHandler(func(req *mockhttp.Request, rsp *mockhttp.Response) {
	//	req.Method = "GET"
	//	req.Endpoint = "/HelloWorld"
	//	req.Host = "www.baidu.com"
	//	rsp.Body = "xxxxxxxxx bbbb"
	//})
	resp, err := http.Get("http://www.baidu.com/hello/xxx")
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}

func testMockXorm() {
	mockdb := gmock.NewMockXORM("example")
	mockdb.RegisterModels(&User{})
	mockdb.ResetAndInit()
	db := mockdb.GetXORMEngine()
	var user User
	_, err := db.Where("id=?", 1).Get(&user)
	if err != nil {
		panic(err)
	}
	fmt.Println(user)
}

func testMockDocker() {
	mock := mockdocker.NewMockDockerService()
	//defer mock.Destroy()

	err := mock.InitContainer(func(opts *docker.CreateContainerOptions) {
		//opts.Name = "alpine"
		opts.Config.Image = "alpine"
		opts.Config.AttachStdout = true
		opts.Config.Cmd = []string{"sleep", "100"}

	})

	fmt.Println(err)

}
