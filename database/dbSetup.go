package database

import (
	"context"
	"database/sql"
	"time"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// LoadDBInstance... for connection to sql server
func (database *Database) LoadDBInstance() {

	db, err := sql.Open("mysql", database.Config.DBConnectionString)
	if err != nil {
		log.Fatal("Error creating database connection: %s", err.Error())
	}

	ctx := context.Background()
	if err = db.PingContext(ctx); err != nil {
		log.Fatal("Database connection closed. Error > %s", err.Error())
	}

	db.SetConnMaxLifetime(time.Minute * 30)
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(5)

	database.DB = db
	database.Logger.Info("Database connection successful!")

	return
}
