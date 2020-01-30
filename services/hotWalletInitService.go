package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// func initHotWallet(repository database.IRepository, logger *utility.Logger, configuration config.Data, userID uuid.UUID, symbol string, serviceErr interface{}) {

// 	//APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))

// 	var externalServiceErr model.ServicesRequestErr

// 	// 1. First check if hot wallet addresses haven't been generated yet.
// 	// 2. If not yet, generate a new address per each asset supported.

// 	for _, asset := range config.SupportedAssets {

// 		address, err := GenerateAddress(logger, configuration, uuid.NewV4(), asset, externalServiceErr)

// 		if err != nil {

// 		}

// 		fmt.Printf("Address generated %s", address)

// 		//repository.Create()

// 	}

// }

// GetHotWalletAddressFor ... Get the Bundle hot wallet address corresponding to a certain asset
func GetHotWalletAddressFor(DB *gorm.DB, logger *utility.Logger, config Config.Data, asseSymbol string) (string, error) {
	hotWallet := dto.HotWalletAsset{}
	externalServiceErr := model.ServicesRequestErr{}
	serviceID, _ := uuid.FromString(config.ServiceID)

	if err := DB.Where(dto.HotWalletAsset{AssetSymbol: asseSymbol}).First(&hotWallet).Error; err != nil {
		if err.Error() != utility.SQL_404 {
			return "", err
		}
		address, err := GenerateAddress(logger, config, serviceID, asseSymbol, &externalServiceErr)
		if err != nil {
			return "", err
		}
		return address, nil
	}

	return "", nil
}
