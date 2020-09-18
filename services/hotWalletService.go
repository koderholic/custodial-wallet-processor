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

//HotWalletService object
type HotWalletService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewHotWalletService(cache *utility.MemoryCache, config Config.Data) *HotWalletService {
	baseService := HotWalletService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

func (service *HotWalletService) InitHotWallet(cache *utility.MemoryCache, DB *gorm.DB, config Config.Data) error {

	supportedAssets := []model.Denomination{}
	coinTypeToAddrMap := map[int64]string{}
	externalServiceErr := dto.ExternalServicesRequestErr{}
	serviceID, _ := uuid.FromString(config.ServiceID)
	address := ""
	var err error

	if err := DB.Order("created_at", true).Find(&supportedAssets).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return err
		}
	}

	for _, asset := range supportedAssets {
		if strings.EqualFold(asset.WithdrawActivity, utility.ACTIVE) {

			address, err = service.GetHotWalletAddressFor(cache, DB, config, asset.AssetSymbol)
			if err != nil {
				logger.Error("Error with getting hot wallet address for %s : %s", asset.AssetSymbol, err)
				return err
			}

			if address != "" {
				coinTypeToAddrMap[asset.CoinType] = address
				continue
			}

			if coinTypeToAddrMap[asset.CoinType] != "" {
				address = coinTypeToAddrMap[asset.CoinType]
			} else {
				KeyManagementService := NewKeyManagementService(service.Cache, service.Config)
				address, err = KeyManagementService.GenerateAddressWithoutSub(cache, config, serviceID, asset.AssetSymbol, &externalServiceErr)
				if err != nil {
					return err
				}
				coinTypeToAddrMap[asset.CoinType] = address
			}

			if err := DB.Create(&model.HotWalletAsset{Address: address, AssetSymbol: asset.AssetSymbol}).Error; err != nil {
				logger.Error("Error with creating hot wallet asset record %s : %s", asset.AssetSymbol, err)
			}
		}
	}

	return nil

}

// GetHotWalletAddressFor ... Get the Bundle hot wallet address corresponding to a certain asset
func (service *HotWalletService) GetHotWalletAddressFor(cache *utility.MemoryCache, DB *gorm.DB, config Config.Data, asseSymbol string) (string, error) {
	hotWallet := model.HotWalletAsset{}

	if err := DB.Where(model.HotWalletAsset{AssetSymbol: asseSymbol}).First(&hotWallet).Error; err != nil {
		if err.Error() != errorcode.SQL_404 {
			return "", err
		}
		return "", nil
	}

	return hotWallet.Address, nil
}
