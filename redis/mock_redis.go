package redis

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

type MockRedisServer struct {
	port        int `json:"port"`
	redisServer *miniredis.Miniredis
}

func NewMockRedisServer() *MockRedisServer {

	svc := MockRedisServer{
		port:        23436,
		redisServer: miniredis.NewMiniRedis(),
	}
	err := svc.redisServer.Start()
	if err != nil {
		panic(err)
	}
	return &svc
}

func (m *MockRedisServer) GetRedisClient() *redis.Client {
	var opt redis.Options
	opt.Addr = m.redisServer.Addr()
	return redis.NewClient(&opt)
}
