package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

func InitHotWallet(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data) error {

	supportedAssets := []dto.Denomination{}

	if err := DB.Find(&supportedAssets).Error; err != nil {
		if err.Error() != utility.SQL_404 {
			return err
		}
	}

	for _, asset := range supportedAssets {
		address, err := GetHotWalletAddressFor(cache, DB, logger, config, asset.AssetSymbol)
		if err != nil {
			logger.Error("Error with getting hot wallet address for %s : %s", asset.AssetSymbol, err)
			return err
		}
		if address != "" {
			if err := DB.Create(&dto.HotWalletAsset{Address: address, AssetSymbol: asset.AssetSymbol}).Error; err != nil {
				logger.Error("Error with creating hot wallet asset record %s : %s", asset.AssetSymbol, err)
			}
		}

	}

	return nil

}

// GetHotWalletAddressFor ... Get the Bundle hot wallet address corresponding to a certain asset
func GetHotWalletAddressFor(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data, asseSymbol string) (string, error) {
	hotWallet := dto.HotWalletAsset{}
	externalServiceErr := model.ServicesRequestErr{}
	serviceID, _ := uuid.FromString(config.ServiceID)

	if err := DB.Where(dto.HotWalletAsset{AssetSymbol: asseSymbol}).First(&hotWallet).Error; err != nil {
		if err.Error() != utility.SQL_404 {
			return "", err
		}
		address, err := GenerateAddress(cache, logger, config, serviceID, asseSymbol, &externalServiceErr)
		if err != nil {
			return "", err
		}
		return address, nil
	}

	return "", nil
}
