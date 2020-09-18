package services

import (
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

//SharedAddressService object
type SharedAddressService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewSharedAddressService(cache *utility.MemoryCache, config Config.Data) *SharedAddressService {
	baseService := SharedAddressService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

// InitSharedAddress ... Initialize the shared addresses for each assets
func (service *SharedAddressService) InitSharedAddress(cache *utility.MemoryCache, DB *gorm.DB, config Config.Data) error {

	supportedAssets := []model.Denomination{}
	externalServiceErr := dto.ExternalServicesRequestErr{}
	userID, _ := uuid.FromString(utility.SHARED_ADDRESS_ID)
	address := ""
	var err error

	if err := DB.Order("is_token asc").Where(&model.Denomination{RequiresMemo: true}).Find(&supportedAssets).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return err
		}
	}

	for _, asset := range supportedAssets {
		if strings.EqualFold(asset.DepositActivity, utility.ACTIVE) {
			SharedAddressService := NewSharedAddressService(service.Cache, service.Config)
			address, err = SharedAddressService.GetSharedAddressFor(cache, DB, config, asset.AssetSymbol)
			if err != nil {
				logger.Error("Error with getting shared address for %s : %s", asset.AssetSymbol, err)
				return err
			}

			if address == "" {
				KeyManagementService := NewKeyManagementService(service.Cache, service.Config)
				address, err = KeyManagementService.GenerateAddress(userID, asset.AssetSymbol, asset.CoinType, &externalServiceErr)
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
func (service *SharedAddressService) GetSharedAddressFor(cache *utility.MemoryCache, DB *gorm.DB, config Config.Data, asseSymbol string) (string, error) {
	sharedAddress := model.SharedAddress{}

	if err := DB.Where(model.SharedAddress{AssetSymbol: asseSymbol}).First(&sharedAddress).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return "", err
		}
		return "", nil
	}

	return sharedAddress.Address, nil
}
