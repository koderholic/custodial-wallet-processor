package database

import (
	"context"
	"log"
	"sync"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/mysql"
)

//Database : database struct
type Database struct {
	Logger *utility.Logger
	Config config.Data
	DB     *gorm.DB
}

var (
	once sync.Once
)

// LoadDBInstance... for connection to sql server
func (database *Database) LoadDBInstance() {

	once.Do(func() {
		db, err := gorm.Open("mysql", database.Config.DBConnectionString)
		if err != nil {
			log.Fatal("Error creating database connection: %s", err.Error())
		}

		ctx := context.Background()
		if err = db.DB().PingContext(ctx); err != nil {
			database.Logger.Error("Database connection closed. Error > %s", err.Error())
		}

		db.DB().SetMaxIdleConns(10)
		db.DB().SetMaxOpenConns(100)
		db.DB().SetConnMaxLifetime(time.Hour)

		db.LogMode(true)
		database.DB = db
	})
	database.Logger.Info("Database connection successful!")
}

func (database *Database) CloseDBInstance() {
	database.DB.Close()
}
