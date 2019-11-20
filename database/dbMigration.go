package database

import (
	"wallet-adapter/model"
)

func (database *Database) RunDbMigrations() {
	database.DB.AutoMigrate(&model.Asset{})
}
