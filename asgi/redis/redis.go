package redis

import (
	"log"
	"syscall"
	"time"

	"github.com/garyburd/redigo/redis"
)

// RedisPool is a pool of redis connections
var redisPool *redis.Pool

// CreateRedisPool sets the redis pool to connect to the host.
func CreateRedisPool(host string) {
	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		MaxActive:   int(getOpenFilesLimit() / 3), // Use a third of the openfiles limit for redis connection
		Wait:        true,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", host) },
	}
}

func getOpenFilesLimit() uint64 {
	var rLimit syscall.Rlimit

	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Fatalf("Can not identify limit of open files: %s", err)
	}
	return rLimit.Cur
}
