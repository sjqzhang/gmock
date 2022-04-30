package gmock

import (
	mockdb "github.com/sjqzhang/gmock/db"
	mockhttp "github.com/sjqzhang/gmock/http"
	mockredis "github.com/sjqzhang/gmock/redis"
)

func NewMockHttpServer(mockJSONDir string, allowProxyHosts []string) *mockhttp.MockHttpServer {
	return mockhttp.NewMockHttpServer(mockJSONDir, allowProxyHosts)
}

func NewMockDB(pathToSqlFileName string) *mockdb.MockDB {
	return mockdb.NewMockDB(pathToSqlFileName)
}

func NewMockRedisServer() *mockredis.MockRedisServer {
	return mockredis.NewMockRedisServer()
}
