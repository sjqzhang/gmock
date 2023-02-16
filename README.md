## 说明
> gmock主要是为了简化单元测试而设计。

## 原理
> 对常用的服务本地内存实现。 

## 目标
> 不改变现有代码的情况下，能快速对原有代码进行单元测试。
## 示例
```go
package main

import (
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/sjqzhang/gmock"
	"github.com/sjqzhang/gmock/mockdb"
	"github.com/sjqzhang/gmock/mockdocker"
	_ "gorm.io/driver/mysql"
	gormv2 "gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"time"
	"xorm.io/xorm"
)

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	testMockGORM()
	testMockGORMV2()
	testMockXORM()
	testMockRedis()
	testMockHttpServer()
	testMockDocker()
	testDBUtil()

}

func testMockGORM() {
	var db *gorm.DB
	mockdb.DBType = "mysql"
	mock := gmock.NewMockGORM("example", func(gorm *mockdb.MockGORM) {
		db = gorm.GetGormDB()
	})
	fmt.Println(mock.GetDSN())
	//mock.RegisterModels(&User{})
	mock.InitSchemas(`CREATE TABLE user (
                           id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
                           age int(3) DEFAULT NULL,
                           name varchar(255) DEFAULT NULL COMMENT '名称',
                           PRIMARY KEY (id)
) ENGINE=InnoDB ;`)
	mock.ResetAndInit()

	var user User
	err := db.Where("id=?", 1).Find(&user).Error
	if err != nil {
		panic(err)
	}
	if user.Id != 1 {
		panic(fmt.Errorf("testMockGORM error"))
	}

}

func testDBUtil() {
	util := gmock.NewDBUtil()
	util.RunMySQLServer("test", 33333, false)
	db, err := gorm.Open("mysql", "user:pass@tcp(127.0.0.1:33333)/test?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	sqlText := util.ReadFile("./example/ddl.txt")
	for _, s := range util.ParseSQLText(sqlText) {
		fmt.Println(db.Exec(s))
	}
	fmt.Println(util.QueryListBySQL(db.DB(), "select * from project"))
}

func testMockGORMV2() {
	mockdb.DBType = "mysql"
	var db *gormv2.DB
	mock := gmock.NewMockGORMV2("example", func(orm *mockdb.MockGORMV2) {
		db = orm.GetGormDB()
	})
	//注册模型
	mock.RegisterModels(&User{})
	//初始化数据库及表数据
	mock.ResetAndInit()
	mock.ResetAndInit()
	//db := mock.GetGormDB()
	var user User
	err := db.Where("id=?", 1).Find(&user).Error
	if err != nil {
		panic(err)
	}
	if user.Id != 1 {
		panic(fmt.Errorf("testMockGORMV2 error"))
	}

}

func testMockRedis() {
	server := gmock.NewMockRedisServer(63790)
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
		panic("testMockRedis error")
	}

}

func testMockHttpServer() {
	// 只支持 http 不支持 https
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
	if string(data) != "hello baidu" {
		panic(fmt.Errorf("testMockHttpServer error"))
	}
}

func testMockXORM() {
	var engine *xorm.Engine
	mockdb.DBType = "mysql"
	mock := gmock.NewMockXORM("example", func(orm *mockdb.MockXORM) {
		engine = orm.GetXORMEngine()
	})
	mock.RegisterModels(&User{})

	mock.ResetAndInit()
	db := mock.GetXORMEngine()
	var user User
	_, err := db.Where("id=?", 1).Get(&user)
	if err != nil {
		panic(err)
	}
	if user.Id != 1 {
		panic(fmt.Errorf("testMockXORM error"))
	}
}

func testMockDocker() {
	mock := mockdocker.NewMockDockerService()
	defer mock.Destroy()
	err := mock.InitContainerWithCmd(func(cmd *string) {
		//  注意：容器必须后台运行，否则会挂起，程序不会继续执行,所以要保证你的容器后台运行不退出
		*cmd = "docker run --name some-mysql  -p 3308:3306 -e MYSQL_ROOT_PASSWORD=root -d mysql:5.7"
	})
	fmt.Println(err)
	if !mock.WaitForReady("wget 127.0.0.1:3308 -O -", time.Second*50) {
		panic(fmt.Errorf("mysql start fail"))
	}
	fmt.Println("mysql start success")

}

```

## 生成覆盖率及测试报告

```bash

go test -timeout 0 -covermode=count -coverprofile=coverage.out  -run="^Test" -coverpkg=package1,package2  

go tool cover -html=coverage.out -o coverage.html

```
