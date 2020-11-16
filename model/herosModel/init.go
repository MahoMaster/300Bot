package herosModel

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var err error
var db *sqlx.DB

var c redis.Conn
var redisErr error

func init() {

	db, err = sqlx.Open(`mysql`, `root:root@tcp(127.0.0.1:3306)/300heros?charset=utf8mb4&parseTime=true`)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}

	fmt.Println("300数据库连接成功")
	c, redisErr = redis.Dial("tcp", "127.0.0.1:6379")
	if redisErr != nil {
		fmt.Println("Connect to redis error", redisErr)
		return
	}
	fmt.Println("redis链接成功")
	// defer c.Close()
}
