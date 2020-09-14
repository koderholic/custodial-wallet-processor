package services

import (
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

// CreateUserAsset ... Create given assets for the specified user
func CreateUserAsset(repository database.IUserAssetRepository, assetDenominations []string, userID uuid.UUID) ([]dto.Asset, error) {
	assets := []dto.Asset{}
	for i := 0; i < len(assetDenominations); i++ {
		denominationSymbol := assetDenominations[i]
		denomination := model.Denomination{}

		if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: denominationSymbol, IsEnabled: true}, &denomination); err != nil {
			return assets, err
		}
		balance, _ := decimal.NewFromString("0.00")
		userAssetmodel := model.UserAsset{DenominationID: denomination.ID, UserID: userID, AvailableBalance: balance.String()}
		_ = repository.FindOrCreateAssets(model.UserAsset{DenominationID: denomination.ID, UserID: userID}, &userAssetmodel)

		asset := normalizeUserAsset(userAssetmodel)

		assets = append(assets, asset)
	}
	return assets, nil
}

func normalizeUserAsset(userAssetmodel model.UserAsset) dto.Asset {
	userAsset := dto.Asset{}
	userAsset.ID = userAssetmodel.ID
	userAsset.UserID = userAssetmodel.UserID
	userAsset.AssetSymbol = userAssetmodel.AssetSymbol
	userAsset.AvailableBalance = userAssetmodel.AvailableBalance
	userAsset.Decimal = userAssetmodel.Decimal
	return userAsset
}
