package main

import (
	"context"
	"fmt"
	"gitee.com/chunanyong/zorm"
	"github.com/jinzhu/gorm"
	"github.com/sjqzhang/gmock"
	"github.com/sjqzhang/gmock/mockdb"
	"github.com/sjqzhang/gmock/mockdocker"
	"github.com/sjqzhang/gmock/mockgrpc"
	"github.com/sjqzhang/gmock/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "gorm.io/driver/mysql"
	gormv2 "gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xorm.io/xorm"
)

type User struct {
	Id     int     `json:"id"`
	Name   string  `json:"name"`
	Age    int     `json:"age"`
	Remark *string `json:"remark"`
}

func main() {
	//testMockGORM()
	testMockGORMV2()
	////testMockXORM()
	////testMockZORM()
	////testMockRedis()
	////testMockHttpServer()
	//testMockDocker()
	////testDBUtil()

	//testMockGrpc()

}

func testMockZORM() {
	//var db *zorm.DBDao
	mockdb.DBType = "mysql"
	mock := mockdb.NewMockZORM("example", nil)
	fmt.Println(mock.GetDSN())
	mock.RegisterModels(&User{})
	//db=mock.GetGormDB()
	//a:=mock.GetSqlDB()
	//fmt.Println(a)
	//return

	//	mock.InitSchemas(`CREATE TABLE user (
	//                           id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
	//                           age int(3) DEFAULT NULL,
	//                           name varchar(255) DEFAULT NULL COMMENT '名称',
	//                           PRIMARY KEY (id)
	//) ENGINE=InnoDB ;`)
	mock.ResetAndInit()
	mock.ResetAndInit()
	var user User

	finder := zorm.NewSelectFinder("user").Append("where id=?", 1)
	_, err := zorm.QueryRow(context.Background(), finder, &user)
	if err != nil {
		panic(err)
	}
	if user.Id != 1 {
		panic(fmt.Errorf("testMockZORM error"))
	}

}

func testMockGORM() {
	//	var db *gorm.DB
	//	mockdb.DBType = "mysql"
	//	mock := gmock.NewMockGORM("example", "mock")
	//	db=mock.GetGormDB()
	//	fmt.Println(mock.GetDSN())
	//	mock.RegisterModels(&User{})
	//	mock.InitSchemas(`CREATE TABLE user (
	//                        id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
	//                        age int(3) DEFAULT NULL,
	//                        name varchar(255) DEFAULT NULL COMMENT '名称',
	//remark varchar(255) DEFAULT NULL COMMENT '名称',
	//                        PRIMARY KEY (id)
	//) ENGINE=InnoDB ;`)
	//	mock.ResetAndInit()
	//
	//	mock.DoRecord(mock.GetGormDB())
	//
	//	var user []User
	//	err := db.Where("id=?", 1).Find(&user).Error
	//	if err != nil {
	//		panic(err)
	//	}
	//	if user[0].Id != 1 {
	//		panic(fmt.Errorf("testMockGORM error"))
	//	}
	//
	//	for _, sql := range mock.GetDBUtil().DumpFromRecordInfo(mock.GetSqlDB(), mock.DumpRecorderInfo()) {
	//		fmt.Println(sql)
	//	}

	mockdb.DBType="mysql"
	MockDB := gmock.NewGORMFromDSN("./example", "mysql", "root:mock@tcp(127.0.0.1:63307)/mock?charset=utf8&parseTime=True&loc=Local")

	MockDB.InitSchemas(`CREATE TABLE user (
                        id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
                        age int(3) DEFAULT NULL,
                        name varchar(255) DEFAULT NULL COMMENT '名称',
remark varchar(255) DEFAULT NULL COMMENT '名称',
                        PRIMARY KEY (id)
) ENGINE=InnoDB ;`)
	MockDB.DoRecord(MockDB.GetGormDB())
	MockDB.ResetAndInit()
	var user User
	MockDB.GetGormDB().Where("id=?", 1).Find(&user)
	MockDB.SaveRecordToFile("./db")
	fmt.Println(user)

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

func testMockGrpc() {

	svc := gmock.NewMockGRPC(mockgrpc.WithDirProtocs("example/grpc"), mockgrpc.WithDirStubs("example/grpc/stub"))
	svc.Start()

	conn, err := grpc.Dial(svc.GetAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	c := NewGreeterClient(conn)

	fmt.Println(c.SayHello(context.Background(), &HelloRequest{Name: "world"}))

	//grpcurl -plaintext -d '{"name": "world"}' 127.0.0.1:4770 main.Greeter/SayHello

	//time.Sleep(time.Second * 1000)

}

func testMockGORMV2() {
	//mockdb.DBType = "mysql"
	var db *gormv2.DB
	mock := gmock.NewMockGORMV2("example", "mock")
	db = mock.GetGormDB()
	mock.DoRecord(mock.GetGormDB())
	//注册模型

	mock.RegisterModels(&User{})
	//初始化数据库及表数据
	mock.ResetAndInit()
	mock.ResetAndInit()
	//db := mock.GetGormDB()
	var user []User
	err := db.Where("id=?", 1).Find(&user).Error
	if err != nil {
		panic(err)
	}
	//if user.Id != 1 {
	//	panic(fmt.Errorf("testMockGORMV2 error"))
	//}

	fmt.Println(mock.GetDBUtil().DumpFromRecordInfo(mock.GetSqlDB(), mock.DumpRecorderInfo()))

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
	for i := 0; i < 10; i++ {
		server := gmock.NewMockHttpServer(23435, "./example", []string{"www.baidu.com", "www.jenkins.org"})
		closeFunc := server.InitMockHttpServer()

		//server.SetReqRspHandler(func(req *mockhttp.Request, rsp *mockhttp.Response) {
		//	req.Method = "GET"
		//	req.Endpoint = "/hello/xxx"
		//	req.Host = "www.baidu.com"
		//	rsp.Body = "hello baidu"
		//	rsp.Handler= func(resp http.ResponseWriter, req *http.Request) {
		//		resp.Write([]byte("hello baidu"))
		//	}
		//})
		resp, err := http.Get("http://www.baidu.com/hello/xxx?name=test&aaa=xxx")
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
		closeFunc()
	}
}

func testMockXORM() {
	var engine *xorm.Engine
	mockdb.DBType = "mysql"
	mock := gmock.NewMockXORM("example", func(orm *mockdb.MockXORM) {
		engine = orm.GetXORMEngine()

	})
	mock.DoRecord(mock.GetXORMEngine())
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
	mock.SaveRecordToFile("./example")
}

func testMockDocker() {
	mock := mockdocker.NewMockDockerServiceWithInit(func() <-chan bool {
		c := make(chan bool, 1)
		checker := func() bool {
			count, _ := util.Exec("ps aux|grep -i docker|wc -l")
			cnt, err := strconv.Atoi(strings.TrimSpace(count))
			if err != nil {
				return false
			}
			if cnt > 3 {
				return true
			}
			return false
		}
		timer := time.NewTimer(time.Second * 30)
		for {
			select {
			case <-timer.C:
				c <- false
				goto BREAK
			default:
				if !checker() {
					util.Exec("open /Applications/Docker.app")
					time.Sleep(time.Second * 25)
				} else {
					c <- true
					goto BREAK
				}

			}

		}
		return c
	BREAK:
		return c

	})

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
