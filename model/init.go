package model

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/garyburd/redigo/redis"
)

var db *sqlx.DB
var err error
var c redis.Conn
var redisErr error

func init() {
	db, err = sqlx.Open(`mysql`, `root:root@tcp(127.0.0.1:3306)/300bot?charset=utf8mb4&parseTime=true`)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
	//	defer db.Close()
	fmt.Println("数据库连接成功")
	c, redisErr = redis.Dial("tcp", "127.0.0.1:6379")
	if redisErr != nil {
		fmt.Println("Connect to redis error", redisErr)
		return
	}
	fmt.Println("redis链接成功")
	//	defer c.Close()
}
