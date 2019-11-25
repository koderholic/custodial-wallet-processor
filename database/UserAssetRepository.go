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
			ErrType: utility.INPUTERROR,
			Err:     err,
		}
	}
	return nil
}

// GetAssetsByUserID ...
func (repo *UserAssetRepository) GetAssetsByUserID(id, model interface{}) error {
	if err := repo.DB.Table("user_balances").Select("user_balances.id, user_balances.user_id, user_balances.available_balance, user_balances.reserved_balance, user_balances.asset_id, user_balances.created_at, user_balances.updated_at,assets.name, assets.symbol, assets.token_type, assets.decimal, assets.is_enabled").Joins("left join assets on assets.id = user_balances.asset_id").Where("user_balances.user_id = ?", id).Scan(model).Error; err != nil {
		return utility.AppError{
			ErrType: utility.INPUTERROR,
			Err:     err,
		}
	}
	return nil
}
