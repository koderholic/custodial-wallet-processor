package database

import (
	"fmt"
	"wallet-adapter/utility"
)

// IUserAssetRepository ...
type IUserAssetRepository interface {
	IRepository
	GetAssetsByUserID(id, model interface{}) error
}

// UserAssetRepository ...
type UserAssetRepository struct {
	BaseRepository
}

// GetAssetsByUserID ...
func (repo *UserAssetRepository) GetAssetsByUserID(id, model interface{}) error {
	if err := repo.DB.Select("assets.symbol,user_balances.*").Joins("left join assets ON assets.id = user_balances.asset_id").Where(id).Find(model).Error; err != nil {
		repo.Logger.Error("Error with repository GetAssetsByUserID %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	fmt.Printf("model > %+v > %s", model, repo.DB.Select("assets.symbol,user_balances.*").Joins("left join assets ON assets.id = user_balances.asset_id").Where(id).Find(model).Error)
	return nil
}
