package database

import (
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
)

// IUserAssetRepository ...
type IUserAssetRepository interface {
	IRepository
	GetAssetsByID(id, model interface{}) error
	Db() *gorm.DB
}

// UserAssetRepository ...
type UserAssetRepository struct {
	BaseRepository
}

// GetAssetsByID ...
func (repo *UserAssetRepository) GetAssetsByID(id, model interface{}) error {
	if err := repo.DB.Select("denominations.symbol, denominations.decimal,user_balances.*").Joins("inner join denominations ON denominations.id = user_balances.denomination_id").Where(id).Find(model).Error; err != nil {
		repo.Logger.Error("Error with repository GetAssetsByID %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}

func (repo *UserAssetRepository) Db() *gorm.DB {
	return repo.DB
}
