package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/sjqzhang/gmock"
	"github.com/sjqzhang/gmock/mockdocker"
	"io/ioutil"
	"net/http"
	"time"
)

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	//
	testMockGORM()
	testMockGORMV2()
	testMockXORM()
	testMockRedis()
	testMockHttpServer()
	testMockDocker()

}

func Fetch(db *sql.DB, sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err :=  db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cys, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for rows.Next() {
		l := len(cys)
		vals := make([]interface{}, l)
		valPtr := make([]interface{}, l)
		for i, _ := range vals {
			valPtr[i] = &vals[i]
		}
		row := make(map[string]interface{}, l)
		rows.Scan(valPtr...)
		for i, c := range cys {
			row[c.Name()] = vals[i]
		}
		result=append(result,row)
	}
	return result, nil
}

func testMockGORM() {
	mockdb := gmock.NewMockGORM("example")
	mockdb.RegisterModels(&User{})
	mockdb.ResetAndInit()
	db := mockdb.GetGormDB()
	var user User
	err := db.Where("id=?", 1).Find(&user).Error
	if err != nil {
		panic(err)
	}
	fmt.Println(user)
//	result, err := Fetch(db.DB(), `WITH tables AS (SELECT name tableName, sql
//FROM sqlite_master WHERE type = 'table' AND tableName NOT LIKE 'sqlite_%')
//SELECT fields.name, fields.type, tableName
//FROM tables CROSS JOIN pragma_table_info(tables.tableName) fields`)
//	data,err:=json.Marshal(result)
//	fmt.Println(string(data),err)

}
func testMockGORMV2() {
	mockdb := gmock.NewMockGORMV2("example")
	//注册模型
	mockdb.RegisterModels(&User{})
	//初始化数据库及表数据
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
	fmt.Println(string(data))
}

func testMockXORM() {
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
	defer mock.Destroy()
	err := mock.InitContainerWithCmd(func(cmd *string) {
		//  注意：容器必须后台运行，否则会挂起，程序不会继续执行
		*cmd = "docker run --name some-mysql  -p 3308:3306 -e MYSQL_ROOT_PASSWORD=root -d mysql:5.7"
	})
	fmt.Println(err)
	if !mock.WaitForReady("wget 127.0.0.1:3308 -O -", time.Second*50) {
		panic(fmt.Errorf("mysql start fail"))
	}
	fmt.Println("mysql start success")

}
