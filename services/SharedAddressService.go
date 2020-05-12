package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// InitSharedAddress ... Initialize the shared addresses for each assets
func InitSharedAddress(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data) error {

	supportedAssets := []model.Denomination{}
	externalServiceErr := dto.ServicesRequestErr{}
	userID, _ := uuid.FromString(utility.SHARED_ADDRESS_ID)
	address := ""
	var err error

	if err := DB.Order("is_token asc").Where(&model.Denomination{RequiresMemo: true}).Find(&supportedAssets).Error; err != nil {
		if err.Error() != utility.SQL_404 {
			return err
		}
	}

	for _, asset := range supportedAssets {

		address, err = GetSharedAddressFor(cache, DB, logger, config, asset.AssetSymbol)
		if err != nil {
			logger.Error("Error with getting shared address for %s : %s", asset.AssetSymbol, err)
			return err
		}

		if address == "" {
			address, err = GenerateAddress(cache, logger, config, userID, asset.AssetSymbol, &externalServiceErr)
			if err != nil {
				return err
			}

			if err := DB.Create(&model.SharedAddress{UserId: userID, Address: address, AssetSymbol: asset.AssetSymbol, CoinType: asset.CoinType}).Error; err != nil {
				logger.Error("Error with creating shared address for asset %s : %s", asset.AssetSymbol, err)
			}
		}
	}

	return nil

}

// GetSharedAddressFor ... Get the Bundle shared address corresponding to a certain asset
func GetSharedAddressFor(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data, asseSymbol string) (string, error) {
	sharedAddress := model.SharedAddress{}

	if err := DB.Where(model.SharedAddress{AssetSymbol: asseSymbol}).First(&sharedAddress).Error; err != nil {
		if err.Error() != utility.SQL_404 {
			return "", err
		}
		return "", nil
	}

	return sharedAddress.Address, nil
}
