package test

import (
	"time"
	"wallet-adapter/dto"

	uuid "github.com/satori/go.uuid"
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
	recordID := uuid.NewV4()
	dbMockedData := dto.UserAssetBalance{
		UserID:  model.(*dto.UserBalance).UserID,
		AssetID: model.(*dto.UserBalance).AssetID,
	}
	dbMockedData.ID = recordID
	dbMockedData.CreatedAt = time.Now()
	dbMockedData.UpdatedAt = time.Now()
	return nil
}

// GetAssetsByUserID ...
func (repo *UserAssetRepository) GetAssetsByUserID(id, model interface{}) error {
	recordID := uuid.NewV4()
	dbMockedData := dto.UserAssetBalance{
		UserID:  model.(*dto.UserBalance).UserID,
		AssetID: model.(*dto.UserBalance).AssetID,
	}
	dbMockedData.ID = recordID
	dbMockedData.CreatedAt = time.Now()
	dbMockedData.UpdatedAt = time.Now()
	return nil
}
