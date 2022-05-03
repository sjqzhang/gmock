package gmock

import (
	"github.com/sjqzhang/gmock/mockdb"
	"github.com/sjqzhang/gmock/mockhttp"
	"github.com/sjqzhang/gmock/mockredis"
)

func NewMockHttpServer(mockJSONDir string, allowProxyHosts []string) *mockhttp.MockHttpServer {
	return mockhttp.NewMockHttpServer(mockJSONDir, allowProxyHosts)
}

func NewMockGORM(pathToSqlFileName string) *mockdb.MockGORM {
	return mockdb.NewMockGORM(pathToSqlFileName)
}

func NewMockGORMV2(pathToSqlFileName string) *mockdb.MockGORMV2 {
	return mockdb.NewMockGORMV2(pathToSqlFileName)
}

func NewMockRedisServer() *mockredis.MockRedisServer {
	return mockredis.NewMockRedisServer()
}

func NewMockXORM(pathToSqlFileName string) *mockdb.MockXORM {
	return mockdb.NewMockXORM(pathToSqlFileName)
}
