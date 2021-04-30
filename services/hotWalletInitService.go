package services

import (
	uuid "github.com/satori/go.uuid"
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
)

func InitHotWallet(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data) error {

	networks := []model.Network{}
	coinTypeToAddrMap := map[int64]string{}
	externalServiceErr := dto.ServicesRequestErr{}
	serviceID, _ := uuid.FromString(config.ServiceID)
	address := ""
	var err error

	if err := DB.Order("asset_symbol", true).Order("is_token", true).Find(&networks).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return err
		}
	}

	for _, networkAsset := range networks {
		if strings.EqualFold(networkAsset.WithdrawActivity, utility.ACTIVE) {

			address, err = GetHotWalletAddressFor(cache, DB, logger, config, networkAsset.AssetSymbol, networkAsset.Network)
			if err != nil {
				logger.Error("Error with getting hot wallet address for %s : %s on network : %s", networkAsset.AssetSymbol, networkAsset.Network, err)
				return err
			}

			if address != "" {
				coinTypeToAddrMap[networkAsset.CoinType] = address
				continue
			}

			if coinTypeToAddrMap[networkAsset.CoinType] != "" {
				address = coinTypeToAddrMap[networkAsset.CoinType]
			} else {
				address, err = GenerateAddressWithoutSub(cache, logger, config, serviceID, networkAsset.NativeAsset, networkAsset.Network, &externalServiceErr)
				if err != nil {
					return err
				}
				coinTypeToAddrMap[networkAsset.CoinType] = address
			}

			if err := DB.Create(&model.HotWalletAsset{Address: address, AssetSymbol: networkAsset.AssetSymbol, Network: networkAsset.Network}).Error; err != nil {
				logger.Error("Error with creating hot wallet asset record %s : %s on network : %s", networkAsset.AssetSymbol, networkAsset.Network, err)
			}
		}
	}

	return nil

}

// GetHotWalletAddressFor ... Get the Bundle hot wallet address corresponding to a certain asset
func GetHotWalletAddressFor(cache *utility.MemoryCache, DB *gorm.DB, logger *utility.Logger, config Config.Data, asseSymbol, network string) (string, error) {
	hotWallet := model.HotWalletAsset{}

	if err := DB.Where(model.HotWalletAsset{AssetSymbol: asseSymbol, Network: network}).First(&hotWallet).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return "", err
		}
		return "", nil
	}

	return hotWallet.Address, nil
}
