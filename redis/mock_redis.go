package redis

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	redigo "github.com/gomodule/redigo/redis"
	"time"
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

func (m *MockRedisServer) FastForward(duration time.Duration) {
	m.redisServer.FastForward(duration)
}

func (m *MockRedisServer) GetRedisClient() *redis.Client {
	var opt redis.Options
	opt.Addr = m.redisServer.Addr()
	return redis.NewClient(&opt)
}

func (m *MockRedisServer) GetRedigoPool() *redigo.Pool {
	var opt redis.Options
	opt.Addr = m.redisServer.Addr()
	return &redigo.Pool{
		MaxActive:   100,
		MaxIdle:     3,
		IdleTimeout: time.Second * 10,
		Dial: func() (redigo.Conn, error) {
			return redigo.Dial("tcp", m.redisServer.Addr())
		},
		TestOnBorrow: func(c redigo.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
