package gmock

import (
	"github.com/sjqzhang/gmock/mockdb"
	"github.com/sjqzhang/gmock/mockhttp"
	"github.com/sjqzhang/gmock/mockredis"
)

func NewMockHttpServer(mockJSONDir string, allowProxyHosts []string) *mockhttp.MockHttpServer {
	return mockhttp.NewMockHttpServer(mockJSONDir, allowProxyHosts)
}

func NewMockDB(pathToSqlFileName string) *mockdb.MockDB {
	return mockdb.NewMockDB(pathToSqlFileName)
}

func NewMockDBV2(pathToSqlFileName string) *mockdb.MockDBV2 {
	return mockdb.NewMockDBV2(pathToSqlFileName)
}

func NewMockRedisServer() *mockredis.MockRedisServer {
	return mockredis.NewMockRedisServer()
}
