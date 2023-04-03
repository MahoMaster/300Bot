package immortalModel

import (
	"log"

	"github.com/garyburd/redigo/redis"
)

func SetRedis(key string, value string, expire int) {
	c.Do("SET", key, value)
	c.Do("expire", key, expire)
}

func DelRedis(key string) {
	c.Do("DEL", key)
}

func GetRedis(key string) (bool, string) {
	exit, err := redis.Bool(c.Do("EXISTS", key))
	if err != nil {

		log.Println(err)
		return false, ""
	} else {
		if exit {
			data, _ := redis.String(c.Do("GET", key))
			return true, data
		} else {
			return false, ""
		}
	}
}
