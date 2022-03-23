package redis

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gomodule/redigo/redis"
)

var redis_pool = newPool()

func newPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
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

func push(svc, ip string, reqs int64) {
	conn := redis_pool.Get()
	defer conn.Close()

	// TODO: change the code to get the password from the env vars
	if _, err := conn.Do("AUTH", "p@ssw0rd"); err != nil {
		log.Panic(err)
	}
	_, err := conn.Do("hset", svc, ip, reqs)
	if err != nil {
		log.Panic(err)
	}
}

func retrieve(svc string) (map[string]int, error) {
	conn := redis_pool.Get()
	defer conn.Close()

	// TODO: change the code to get the password from the env vars
	if _, err := conn.Do("AUTH", "p@ssw0rd"); err != nil {
		log.Panic(err) // need to change to retry
	}

	vals, err := conn.Do("hgetall", svc)
	if err != nil {
		return nil, err
	}

	var res map[string]int
	var k string

	for i, v := range vals.([]interface{}) {
		val := fmt.Sprintf("%s", v)
		if i%2 == 1 {
			v_int, _ := strconv.Atoi(val)
			res[k] = v_int
		} else {
			k = val
		}
	}

	return res, nil
}
