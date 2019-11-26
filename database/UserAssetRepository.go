package database

import (
	"wallet-adapter/utility"
)

// IUserAssetRepository ...
type IUserAssetRepository interface {
	IRepository
	GetAssetsByUserID(id, model interface{}) error
	FindOrCreateUserAsset(checkExistOrUpdate interface{}, model interface{}) error
}

// UserAssetRepository ...
type UserAssetRepository struct {
	BaseRepository
}

// FindOrCreateUserAsset ...
func (repo *UserAssetRepository) FindOrCreateUserAsset(checkExistOrUpdate interface{}, model interface{}) error {
	if err := repo.DB.FirstOrCreate(model, checkExistOrUpdate).Error; err != nil {
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}

// GetAssetsByUserID ...
func (repo *UserAssetRepository) GetAssetsByUserID(id, model interface{}) error {
	if err := repo.DB.Select("assets.*,user_balances.*").Joins("left join assets ON assets.id = user_balances.asset_id").Where(id).Find(model).Error; err != nil {
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}
