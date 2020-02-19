package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/utility"

	"github.com/go-redis/redis/v7"
	// "github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/mysql"
)

//Database : database struct
type Database struct {
	Logger      *utility.Logger
	Config      config.Data
	DB          *gorm.DB
	RedisClient *redis.Client
}

var (
	once sync.Once
)

// LoadDBInstance... for connection to sql server
func (database *Database) LoadDBInstance() {

	once.Do(func() {
		DBConnectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", database.Config.DBUser, database.Config.DBPassword, database.Config.DBHost, database.Config.DBName)
		//DBConnectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", "root", "password","127.0.0.1", "wallet-adapter")
		db, err := gorm.Open("mysql", DBConnectionString)
		if err != nil {
			log.Fatal("Error creating database connection: %s", err.Error())
		}

		ctx := context.Background()
		if err = db.DB().PingContext(ctx); err != nil {
			database.Logger.Error("Database connection closed. Error > %s", err.Error())
		}

		db.DB().SetMaxIdleConns(25)
		db.DB().SetMaxOpenConns(50)
		db.DB().SetConnMaxLifetime(time.Second * 300)

		db.LogMode(true)
		database.DB = db
	})
	database.Logger.Info("Database connection successful!")
}

// CloseDBInstance ...
func (database *Database) CloseDBInstance() {
	database.DB.Close()
}
