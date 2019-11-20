package database

import (
	"wallet-adapter/model"
)

func (database *Database) RunDbMigrations() {
	database.DB.AutoMigrate(&model.Asset{}, &model.BatchRequest{}, &model.ChainTransaction{}, &model.Transaction{}, &model.UserAddress{}, &model.UserBalance{})
}
