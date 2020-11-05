package services

import (
	"github.com/jinzhu/gorm"
	"wallet-adapter/model"
	"wallet-adapter/utility"
)

func InitFloatParams(DB *gorm.DB, logger *utility.Logger) error {

	var hotWalletAssets []model.HotWalletAsset
	if err := DB.Find(&hotWalletAssets).Error; err != nil {
		return err
	}

	for _, asset := range hotWalletAssets {
		floatParam := model.FloatManagerParam{
			AssetSymbol: asset.AssetSymbol,
			MinPercentMaxUserBalance     : 0.6,
			MaxPercentMaxUserBalance      : 0.8,
			MinPercentTotalUserBalance    : 0.3,
			AveragePercentTotalUserBalance :  0.4,
			MaxPercentTotalUserBalance     : 0.6,
			PercentMinimumTriggerLevel  : 0.8,
			PercentMaximumTriggerLevel     : 0.3,
		}
		if err := DB.Where(model.FloatManagerParam{AssetSymbol: asset.AssetSymbol}).FirstOrCreate(&floatParam).Error; err != nil {
			logger.Error("Error with creating float params for asset %s : %s", asset.AssetSymbol, err)
		}
	}
	return nil
}
