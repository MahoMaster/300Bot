package immortalModel

import (
	"300Bot/conf"
	"time"

	// "fmt"
	"log"

	"github.com/gomodule/redigo/redis"
	//_ "github.com/go-sql-driver/mysql"
	//"github.com/jmoiron/sqlx"
	// "gorm.io/driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB
var err error
var c redis.Conn
var redisErr error

func init() {
	dsn := conf.Config.DatabaseUser + `:` + conf.Config.DatabasePassword + `@tcp(` + conf.Config.DatabaseHost + `)/` + conf.Config.ImmortalbaseName + `?charset=utf8mb4&parseTime=true`
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	// db, err = sqlx.Open(`mysql`, )
	if err != nil {
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	if err = sqlDB.Ping(); err != nil {
		panic(err)
	}
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetConnMaxLifetime(time.Hour)
	// db.SetMaxOpenConns(8)
	// db.SetMaxIdleConns(4)
	//	def er db.Close()
	log.Println("数据库连接成功")

	c, redisErr = redis.Dial("tcp", "127.0.0.1:6379")
	if redisErr != nil {
		log.Println("Connect to redis error", redisErr)
		return
	}
	log.Println("redis链接成功")

}
