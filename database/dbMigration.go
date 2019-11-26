package database

import (
	"wallet-adapter/model"
)

// RunDbMigrations ... This creates corresponding tables for models on the db and watches the model for field additions and not field changes
func (database *Database) RunDbMigrations() {
	database.DB.AutoMigrate(&model.Asset{}, &model.BatchRequest{}, &model.ChainTransaction{}, &model.Transaction{}, &model.UserAddress{}, &model.UserBalance{})
	database.DB.Model(&model.UserBalance{}).AddForeignKey("asset_id", "assets(id)", "CASCADE", "CASCADE")
}
