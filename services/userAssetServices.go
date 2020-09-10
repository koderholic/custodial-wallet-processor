package services

import (
	"fmt"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/shopspring/decimal"
)

type IServiceRepository interface {
}

func CreateAsset(repository IRepository, denominations []string) {

	// Create user asset record for each given denomination
	for i := 0; i < len(denominations); i++ {
		denominationSymbol := requestData.Assets[i]
		denomination := model.Denomination{}

		if err := controller.Repository.GetByFieldName(&model.Denomination{AssetSymbol: denominationSymbol, IsEnabled: true}, &denomination); err != nil {
			if err.Error() == errorcode.SQL_404 {
				ReturnError(responseWriter, "CreateUserAssets", http.StatusNotFound, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", denominationSymbol)), controller.Logger)
				return
			}
			ReturnError(responseWriter, "CreateUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
			return
		}
		balance, _ := decimal.NewFromString("0.00")
		userAssetmodel := model.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID, AvailableBalance: balance.String()}
		_ = controller.Repository.FindOrCreateAssets(model.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID}, &userAssetmodel)

		responseData.Assets = append(responseData.Assets, userAsset)
	}

}

func normalizeAsset() {
	userAsset := dto.Asset{}
	userAsset.ID = userAssetmodel.ID
	userAsset.UserID = userAssetmodel.UserID
	userAsset.AssetSymbol = userAssetmodel.AssetSymbol
	userAsset.AvailableBalance = userAssetmodel.AvailableBalance
	userAsset.Decimal = userAssetmodel.Decimal
}
