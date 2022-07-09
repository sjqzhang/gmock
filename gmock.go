package gmock

import (
	"github.com/sjqzhang/gmock/mockdb"
	"github.com/sjqzhang/gmock/mockhttp"
	"github.com/sjqzhang/gmock/mockredis"
	"github.com/sjqzhang/gmock/util"
	"github.com/sjqzhang/requests"
	"net/http"
	"net/http/httptest"
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

func NewGORMFromDSN(pathToSqlFileName string, dbType string, dsn string) *mockdb.MockGORM {
	return mockdb.NewGORMFromDSN(pathToSqlFileName, dbType, dsn)
}

func NewGORMV2FromDSN(pathToSqlFileName string, dbType string, dsn string) *mockdb.MockGORMV2 {
	return mockdb.NewGORMV2FromDSN(pathToSqlFileName, dbType, dsn)
}

func NewXORMFromDSN(pathToSqlFileName string, dbType string, dsn string) *mockdb.MockXORM {
	return mockdb.NewXORMFromDSN(pathToSqlFileName, dbType, dsn)
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

func NewRecorder() *httptest.ResponseRecorder {
	return requests.NewRecorder()
}

func NewRequestForTest(method, origurl string, args ...interface{}) (*http.Request, error) {
	return requests.NewRequestForTest(method, origurl, args...)
}


