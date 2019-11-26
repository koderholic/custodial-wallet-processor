package database

import (
	"wallet-adapter/utility"
)

// IAssetRepository ...
type IAssetRepository interface {
	IRepository
	GetSupportedCrypto(model interface{}) error
}

// AssetRepository ...
type AssetRepository struct {
	BaseRepository
}

// GetSupportedCrypto ...
func (repo *AssetRepository) GetSupportedCrypto(model interface{}) error {
	if err := repo.DB.Where("is_enabled = ? ", true).Find(model).Error; err != nil {
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}
