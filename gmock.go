package gmock

import (
	"github.com/sjqzhang/requests"
	"github.com/sjqzhang/gmock/mockdb"
	"github.com/sjqzhang/gmock/mockhttp"
	"github.com/sjqzhang/gmock/mockredis"
	"github.com/sjqzhang/gmock/util"
)

func NewMockHttpServer(httpServerPort int, mockJSONDir string, allowProxyHosts []string) *mockhttp.MockHttpServer {
	return mockhttp.NewMockHttpServer(httpServerPort, mockJSONDir, allowProxyHosts)
}

func NewMockGORM(pathToSqlFileName string, resetHandler func(orm *mockdb.MockGORM)) *mockdb.MockGORM {
	return mockdb.NewMockGORM(pathToSqlFileName, resetHandler)
}

func NewMockGORMV2(pathToSqlFileName string, resetHandler func(orm *mockdb.MockGORMV2)) *mockdb.MockGORMV2 {
	return mockdb.NewMockGORMV2(pathToSqlFileName, resetHandler)
}

func NewMockRedisServer(port int) *mockredis.MockRedisServer {
	return mockredis.NewMockRedisServer(port)
}

func NewMockXORM(pathToSqlFileName string, resetHandler func(orm *mockdb.MockXORM)) *mockdb.MockXORM {
	return mockdb.NewMockXORM(pathToSqlFileName, resetHandler)
}

func NewDBUtil() *util.DBUtil {
	return util.NewDBUtil()
}

func Get(origurl string, args ...interface{}) (resp *requests.Response, err error) {

	return requests.Get(origurl, args...)
}

func Post(origurl string, args ...interface{}) (resp *requests.Response, err error) {
	return requests.Post(origurl, args...)
}
