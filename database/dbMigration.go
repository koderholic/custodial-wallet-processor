package database

import (
	"wallet-adapter/dto"
)

// RunDbMigrations ... This creates corresponding tables for dtos on the db and watches the dto for field additions
func (database *Database) RunDbMigrations() {
	database.DB.AutoMigrate(&dto.Asset{}, &dto.BatchRequest{}, &dto.ChainTransaction{}, &dto.Transaction{}, &dto.UserAddress{}, &dto.UserBalance{})
	database.DB.Model(&dto.UserBalance{}).AddForeignKey("asset_id", "assets(id)", "CASCADE", "CASCADE")
}
