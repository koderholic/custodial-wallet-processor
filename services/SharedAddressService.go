package services

import (
	"net/http"
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

//SharedAddressService object
type SharedAddressService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewSharedAddressService(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr *dto.ExternalServicesRequestErr) *SharedAddressService {
	baseService := SharedAddressService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      serviceErr,
	}
	return &baseService
}

// InitSharedAddress ... Initialize the shared addresses for each assets
func (service *SharedAddressService) InitSharedAddress(DB *gorm.DB) error {

	supportedAssets := []model.Denomination{}
	userID, _ := uuid.FromString(constants.SHARED_ADDRESS_ID)
	address := ""
	var err error

	if err := DB.Order("is_token asc").Where(&model.Denomination{RequiresMemo: true}).Find(&supportedAssets).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return serviceError(http.StatusInternalServerError, errorcode.SERVER_ERR_CODE, err)
		}
		return serviceError(http.StatusNotFound, errorcode.RECORD_NOT_FOUND, err)
	}

	for _, asset := range supportedAssets {
		if strings.EqualFold(asset.DepositActivity, constants.ACTIVE) {
			address, err = service.GetSharedAddressFor(DB, asset.AssetSymbol)
			if err != nil {
				logger.Error("Error with getting shared address for %s : %s", asset.AssetSymbol, err)
				return err
			}

			if address == "" {
				KeyManagementService := NewKeyManagementService(service.Cache, service.Config, service.Repository, service.Error)
				address, err = KeyManagementService.GenerateAddress(userID, asset.AssetSymbol, asset.CoinType)
				if err != nil {
					return err
				}

				if err := DB.Create(&model.SharedAddress{UserId: userID, Address: address, AssetSymbol: asset.AssetSymbol, CoinType: asset.CoinType}).Error; err != nil {
					logger.Error("Error with creating shared address for asset %s : %s", asset.AssetSymbol, err)
				}
			}
		}
	}

	return nil

}

// GetSharedAddressFor ... Get the Bundle shared address corresponding to a certain asset
func (service *SharedAddressService) GetSharedAddressFor(DB *gorm.DB, asseSymbol string) (string, error) {
	sharedAddress := model.SharedAddress{}

	if err := DB.Where(model.SharedAddress{AssetSymbol: asseSymbol}).First(&sharedAddress).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return "", serviceError(http.StatusInternalServerError, errorcode.SERVER_ERR_CODE, err)
		}
		return "", serviceError(http.StatusNotFound, errorcode.RECORD_NOT_FOUND, err)
	}

	return sharedAddress.Address, nil
}
