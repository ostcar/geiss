package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

// RedisPool is a poll of redis connections
var RedisPool *redis.Pool

func init() {
	RedisPool = newRedisPool(":6379")
}

func newRedisPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}
}
