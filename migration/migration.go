package migration

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pressly/goose"
	Config "wallet-adapter/config"
	"wallet-adapter/utility"
)

// RunDbMigrations ... This creates corresponding tables for dtos on the db and watches the dto for field additions
func RunDbMigrations(logger *utility.Logger, config Config.Data) error {
	DBConnectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", config.DBUser, config.DBPassword, config.DBHost, config.DBName)

	db, err := sql.Open("mysql", DBConnectionString)
	if err != nil {
		logger.Error("Error creating db connection for migration: ", err.Error())
		return err
	}
	defer db.Close()
	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		logger.Error("Database connection interrupted : ", err.Error())
		return err
	}

	// Migrate up to the latest version
	goose.SetDialect("mysql")
	err = goose.Up(db, config.DBMigrationPath)
	if err != nil {
		logger.Error("Error with DB Migration : ", err.Error())
		return err
	}
	return nil
}
