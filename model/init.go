package model

import (
	"300Bot/conf"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB
var err error

// var c redis.Conn
// var redisErr error

func init() {
	db, err = sqlx.Open(`mysql`, conf.Config.DatabaseUser+`:`+conf.Config.DatabasePassword+`@tcp(`+conf.Config.DatabaseHost+`)/`+conf.Config.BotDatabaseName+`?charset=utf8mb4&parseTime=true`)
	if err != nil {
		panic(err)
	}
	// 连接池调优：避免空闲连接被服务端 wait_timeout 掐掉后报 bad connection
	db.SetMaxIdleConns(4)
	db.SetMaxOpenConns(8)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)
	if err = db.Ping(); err != nil {
		panic(err)
	}
	if err = ensureMemoryRawTurnsTable(); err != nil {
		panic(err)
	}
	if err = ensureMemoryRawTurnsNicknameColumn(); err != nil {
		panic(err)
	}

	fmt.Println("数据库连接成功")
	// c, redisErr = redis.Dial("tcp", "127.0.0.1:6379")
	// if redisErr != nil {
	// 	fmt.Println("Connect to redis error", redisErr)
	// 	return
	// }
	// fmt.Println("redis链接成功")
	//	defer c.Close()
}
