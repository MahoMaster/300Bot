package immortalModel

import (
	"errors"
	"log"

	"github.com/garyburd/redigo/redis"
)

func SetRedis(key string, value string, expire int) {
	c.Do("SET", key, value)
	if expire != 0 {
		c.Do("expire", key, expire)
	}
}

func SetRedisExpire(key string, expire int) {
	c.Do("expire", key, expire)
}

func DelRedis(key string) {
	c.Do("DEL", key)
}

func GetRedis(key string) (string, error) {
	exit, err := redis.Bool(c.Do("EXISTS", key))
	if err != nil {

		log.Println(err)
		return "", err
	} else {
		if exit {
			data, _ := redis.String(c.Do("GET", key))
			return data, nil
		} else {
			return "", errors.New("key不存在")
		}
	}
}
