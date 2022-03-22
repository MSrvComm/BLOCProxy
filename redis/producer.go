package redis

import (
	"log"

	"github.com/gomodule/redigo/redis"
)

// var CLIENT = newPool().Get()

func NewPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle: 80,
		MaxActive: 1200,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "redis:6379")
			if err != nil {
				log.Panic(err)
			}
			return c, nil
		},
	}
}

func push(svc, ip string, reqs int64)  {
	
}