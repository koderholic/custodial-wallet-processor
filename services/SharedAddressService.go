package services

import (
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// InitSharedAddress ... Initialize the shared addresses for each assets
func InitSharedAddress(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data) error {

	supportedNetworks := []model.Network{}
	userID, _ := uuid.FromString(utility.SHARED_ADDRESS_ID)
	address := ""
	var err error

	if err := DB.Order("is_token asc").Where(&model.Network{RequiresMemo: true}).Find(&supportedNetworks).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return err
		}
	}

	for _, networkAsset := range supportedNetworks {
		if strings.EqualFold(networkAsset.DepositActivity, utility.ACTIVE) {
			address, err = GetSharedAddressFor(cache, DB, logger, config, networkAsset.AssetSymbol, networkAsset.Network)
			if err != nil {
				logger.Error("Error with getting shared address for %s : %s", networkAsset.AssetSymbol, err)
				return err
			}

			if address == "" {
				AddressService := BaseService{Config: config, Cache: cache, Logger: logger}
				address, err  := AddressService.GenerateAllAddresses(userID, networkAsset.AssetSymbol, networkAsset.CoinType, "", networkAsset.Network)
				if err != nil {
					return err
				}

				if err := DB.Create(&model.SharedAddress{UserId: userID, Address: address[0].Data, AssetSymbol: networkAsset.AssetSymbol, Network: networkAsset.Network, CoinType: networkAsset.CoinType}).Error; err != nil {
					logger.Error("Error with creating shared address for asset %s : %s", networkAsset.AssetSymbol, err)
				}
			}
		}
	}

	return nil

}

// GetSharedAddressFor ... Get the Bundle shared address corresponding to a certain asset
func GetSharedAddressFor(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data, assetSymbol, network string) (string, error) {
	sharedAddress := model.SharedAddress{}

	if err := DB.Where(model.SharedAddress{AssetSymbol: assetSymbol, Network: network}).First(&sharedAddress).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return "", err
		}
		return "", nil
	}

	return sharedAddress.Address, nil
}
