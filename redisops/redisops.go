package redisops

import (
	"fmt"
	"log"
	"strconv"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
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

func Flush() {
	conn := redis_pool.Get()
	defer conn.Close()

	// TODO: change the code to get the password from the env vars
	if _, err := conn.Do("AUTH", "p@ssw0rd"); err != nil {
		log.Panic(err)
	}
	_, err := conn.Do("flushall")
	if err != nil {
		log.Panic(err)
	}
}

// update number of requests in an endpoint
func Push(svc, ip string, reqs int64) {
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

// returns a map of endpoints and open requests to each
func Retrieve(svc string) (map[string]int64, error) {
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

	res := make(map[string]int64)
	var k string

	for i, v := range vals.([]interface{}) {
		val := fmt.Sprintf("%s", v)
		if i%2 == 1 {
			v_int, _ := strconv.Atoi(val)
			res[k] = int64(v_int)
		} else {
			k = val
		}
	}

	// check that keys in res
	// add any backends that are missing
	// with the values from the global struct
	// log.Println("Retrieve, svcMap", globals.Endpoints_g) // debug
	// backends := globals.Endpoints_g[svc]

	// backends_int, _ := globals.Endpoints_g.Load(svc)
	// backends := backends_int.([]string)

	backends, err := controllercomm.GetIps(svc)
	if len(backends) != len(res) {
		for _, k := range backends {
			if _, ok := res[k]; !ok {
				res[k] = 0
			}
		}
	}

	log.Println("Retrieve", res) // debug
	return res, nil
}

func Decr(svc, ip string) {
	conn := redis_pool.Get()
	defer conn.Close()

	// TODO: change the code to get the password from the env vars
	if _, err := conn.Do("AUTH", "p@ssw0rd"); err != nil {
		log.Panic(err)
	}
	_, err := conn.Do("hincrby", svc, ip, -1)
	if err != nil {
		log.Panic(err)
	}
}

func Incr(svc, ip string) {
	conn := redis_pool.Get()
	defer conn.Close()

	// TODO: change the code to get the password from the env vars
	if _, err := conn.Do("AUTH", "p@ssw0rd"); err != nil {
		log.Panic(err)
	}
	_, err := conn.Do("hincrby", svc, ip, 1)
	if err != nil {
		log.Panic(err)
	}
}
