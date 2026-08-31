package model

import (
	"300Bot/conf"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB
var err error

// var c redis.Conn
// var redisErr error

func init() {
	// 纯函数单测旁路：BOT300_TEST=1 时只建连接句柄不 Ping 不建表，
	// 避免依赖 model 的包（如 function/memory）无法离线跑单测；生产环境不设置该变量，fail-fast 语义不变
	db, err = sqlx.Open(`mysql`, conf.Config.DatabaseUser+`:`+conf.Config.DatabasePassword+`@tcp(`+conf.Config.DatabaseHost+`)/`+conf.Config.BotDatabaseName+`?charset=utf8mb4&parseTime=true`)
	if err != nil {
		panic(err)
	}
	// 连接池调优：避免空闲连接被服务端 wait_timeout 掐掉后报 bad connection
	db.SetMaxIdleConns(4)
	db.SetMaxOpenConns(8)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)
	if os.Getenv("BOT300_TEST") == "1" {
		return
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
	if err = ensureMemoryRawTurnsTable(); err != nil {
		panic(err)
	}
	if err = ensureMemoryRawTurnsNicknameColumn(); err != nil {
		panic(err)
	}
	if err = ensureMemoryFallbackTable(); err != nil {
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
