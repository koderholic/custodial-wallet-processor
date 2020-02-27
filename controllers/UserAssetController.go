package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"net/http"
	"time"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

// CreateUserAssets ... Creates all supported crypto asset record on the given user account
func (controller UserAssetController) CreateUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreateUserAssetRequest{}
	responseData := model.UserAssetResponse{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for CreateUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	// Create user asset record for each given denomination
	for i := 0; i < len(requestData.Assets); i++ {
		denominationSymbol := requestData.Assets[i]
		denomination := dto.Denomination{}

		if err := controller.Repository.GetByFieldName(&dto.Denomination{AssetSymbol: denominationSymbol, IsEnabled: true}, &denomination); err != nil {
			controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", err)
			if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
				responseWriter.Header().Set("Content-Type", "application/json")
				if err.Error() == utility.SQL_404 {
					responseWriter.WriteHeader(http.StatusNotFound)
				} else {
					responseWriter.WriteHeader(http.StatusInternalServerError)
				}
				json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset record (%s) could not be created for user. %s", denominationSymbol, utility.GetSQLErr(err.(utility.AppError)))))
				return
			}

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", denominationSymbol)))
			return
		}
		balance, _ := decimal.NewFromString("0.00")
		userAssetDTO := dto.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID, AvailableBalance: balance.String()}
		_ = controller.Repository.FindOrCreate(dto.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID}, &userAssetDTO)
		userAssetDTO.AssetSymbol = denomination.AssetSymbol

		userAsset := model.Asset{}
		userAsset.ID = userAssetDTO.ID
		userAsset.UserID = userAssetDTO.UserID
		userAsset.AssetSymbol = userAssetDTO.AssetSymbol
		userAsset.AvailableBalance = userAssetDTO.AvailableBalance
		userAsset.Decimal = denomination.Decimal

		responseData.Assets = append(responseData.Assets, userAsset)
	}

	controller.Logger.Info("Outgoing response to CreateUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssets ... Get all user asset balance
func (controller UserAssetController) GetUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var userAssets []dto.UserAsset
	responseData := model.UserAssetResponse{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	userID, err := uuid.FromString(routeParams["userId"])
	if err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}
	controller.Logger.Info("Incoming request details for GetUserAssets : userID : %+v", userID)

	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{UserID: userID}, &userAssets); err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))))
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssets request %+v", userAssets)

	for i := 0; i < len(userAssets); i++ {
		userAsset := model.Asset{}
		userAssetDTO := userAssets[i]

		userAsset.ID = userAssetDTO.ID
		userAsset.UserID = userAssetDTO.UserID
		userAsset.AssetSymbol = userAssetDTO.AssetSymbol
		userAsset.AvailableBalance = userAssetDTO.AvailableBalance
		userAsset.Decimal = userAssetDTO.Decimal

		responseData.Assets = append(responseData.Assets, userAsset)
	}
	if len(responseData.Assets) <= 0 {
		responseData.Assets = []model.Asset{}
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssetById... Get user asset balance by id
func (controller UserAssetController) GetUserAssetById(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var userAssets dto.UserAsset
	responseData := model.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssetById request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}
	controller.Logger.Info("Incoming request details for GetUserAssetById : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetID}}, &userAssets); err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssetById request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))))
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssetById request %+v", userAssets)

	responseData.ID = userAssets.ID
	responseData.UserID = userAssets.UserID
	responseData.AssetSymbol = userAssets.AssetSymbol
	responseData.AvailableBalance = userAssets.AvailableBalance
	responseData.Decimal = userAssets.Decimal

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssetByAddress ... Get user asset balance by address
func (controller UserAssetController) GetUserAssetByAddress(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var userAssets dto.UserAsset
	var userAddress dto.UserAddress
	responseData := model.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	address := routeParams["address"]

	controller.Logger.Info("Incoming request details for GetUserAssetByAddress : address : %+v", address)

	if err := controller.Repository.GetByFieldName(&dto.UserAddress{Address: address}, &userAddress); err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssetByAddress request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: userAddress.AssetID}}, &userAssets); err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssetByAddress request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))))
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssetByAddress request %+v", userAssets)

	responseData.ID = userAssets.ID
	responseData.UserID = userAssets.UserID
	responseData.AssetSymbol = userAssets.AssetSymbol
	responseData.AvailableBalance = userAssets.AvailableBalance
	responseData.Decimal = userAssets.Decimal

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// DebitUserAsset ... debit a user asset abalance with the specified value
func (controller UserAssetController) DebitUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for DebitUserAsset : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	assetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// // increment user account by volume
	value := decimal.NewFromFloat(requestData.Value)
	availbal, err := decimal.NewFromString(assetDetails.AvailableBalance)
	previousBalance := assetDetails.AvailableBalance
	currentAvailableBalance := (availbal.Sub(value)).String()
	if err != nil {
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestData.AssetID, err)))
		return
	}

	if availbal.Sub(value).IsNegative() {
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INSUFFICIENT_FUNDS", utility.INSUFFICIENT_FUNDS))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestData.AssetID, err)))
		return
	}

	if err := tx.Model(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance - ?", value)).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{

		InitiatorID:          decodedToken.ServiceID, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      dto.TransactionType.OFFCHAIN,
		TransactionStatus:    dto.TransactionStatus.COMPLETED,
		TransactionTag:       dto.TransactionTag.DEBIT,
		Value:                value.String(),
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	responseData.AssetID = requestData.AssetID
	responseData.Value = transaction.Value
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.TransactionStatus = transaction.TransactionStatus

	controller.Logger.Info("Outgoing response to DebitUserAsset request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) CreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for CreditUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and fetc asset
	assetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// // increment user account by volume
	value := decimal.NewFromFloat(requestData.Value)
	availbal, err := decimal.NewFromString(assetDetails.AvailableBalance)
	previousBalance := assetDetails.AvailableBalance
	currentAvailableBalance := (availbal.Add(value)).String()
	if err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)))
		return
	}

	if err := tx.Model(assetDetails).Updates(dto.UserAsset{AvailableBalance: currentAvailableBalance}).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{

		InitiatorID:          decodedToken.ServiceID, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      dto.TransactionType.OFFCHAIN,
		TransactionStatus:    dto.TransactionStatus.COMPLETED,
		TransactionTag:       dto.TransactionTag.CREDIT,
		Value:                value.String(),
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	responseData.AssetID = requestData.AssetID
	responseData.Value = transaction.Value
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.TransactionStatus = transaction.TransactionStatus

	controller.Logger.Info("Outgoing response to CreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) OnChainCreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.OnChainCreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for OnChainCreditUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and fetc asset
	assetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// // increment user account by volume
	value := decimal.NewFromFloat(requestData.Value)
	availbal, err := decimal.NewFromString(assetDetails.AvailableBalance)
	previousBalance := assetDetails.AvailableBalance
	currentAvailableBalance := (availbal.Add(value)).String()
	if err != nil {
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)))
		return
	}

	if err := tx.Model(&assetDetails).Updates(dto.UserAsset{AvailableBalance: currentAvailableBalance}).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	//save chain tx model first, get id and use that in Transaction model
	chainTransaction := dto.ChainTransaction{
		Status:          *requestData.ChainData.Status,
		TransactionHash: requestData.ChainData.TransactionHash,
		TransactionFee:  requestData.ChainData.TransactionFee,
		BlockHeight:     requestData.ChainData.BlockHeight,
	}

	if err := tx.Create(&chainTransaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	transactionStatus := dto.TransactionStatus.PENDING
	if chainTransaction.Status == true {
		transactionStatus = dto.TransactionStatus.COMPLETED
	} else {
		transactionStatus = dto.TransactionStatus.REJECTED
	}
	// Create transaction record
	transaction := dto.Transaction{

		InitiatorID:          decodedToken.ServiceID, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      dto.TransactionType.ONCHAIN,
		TransactionStatus:    transactionStatus,
		TransactionTag:       dto.TransactionTag.DEPOSIT,
		Value:                value.String(),
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
		OnChainTxId:          chainTransaction.ID,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		controller.Logger.Error("Outgoing response to OnChainCreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	responseData.AssetID = requestData.AssetID
	responseData.Value = transaction.Value
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.TransactionStatus = transaction.TransactionStatus

	controller.Logger.Info("Outgoing response to OnChainCreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// InternalTransfer ... transfer between two users
func (controller UserAssetController) InternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.InternalTransferRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for InternalTransfer : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	initiatorAssetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.InitiatorAssetId}}, &initiatorAssetDetails); err != nil {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}
	recipientAssetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.RecipientAssetId}}, &recipientAssetDetails); err != nil {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Ensure transfer cannot be done to self
	if requestData.InitiatorAssetId == requestData.RecipientAssetId {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", utility.NON_MATCHING_DENOMINATION)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.TRANSFER_TO_SELF))
		return
	}

	// Check if the denomination in the transction request is same for initiator and recipient
	if initiatorAssetDetails.DenominationID != recipientAssetDetails.DenominationID {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", utility.NON_MATCHING_DENOMINATION)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.NON_MATCHING_DENOMINATION))
		return
	}

	// Increment user account by volume
	value := decimal.NewFromFloat(requestData.Value)
	availbal, err := decimal.NewFromString(initiatorAssetDetails.AvailableBalance)
	previousBalance := initiatorAssetDetails.AvailableBalance
	currentAvailableBalance := (availbal.Sub(value)).String()
	if err != nil {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("Transfer could not be completed :  %s", err)))
		return
	}

	if availbal.Sub(value).IsNegative() {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("%s", "User asset do not have sufficient balance for this operation")))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("Transfer could not be completed :  %s", err)))
		return
	}

	// Debit Inititor
	if err := tx.Model(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: initiatorAssetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance - ?", value)).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Credit recipient
	if err := tx.Model(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: recipientAssetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance + ?", value)).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{
		InitiatorID:          initiatorAssetDetails.ID,
		RecipientID:          recipientAssetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      dto.TransactionType.OFFCHAIN,
		TransactionStatus:    dto.TransactionStatus.COMPLETED,
		TransactionTag:       dto.TransactionTag.TRANSFER,
		Value:                value.String(),
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	responseData.AssetID = requestData.InitiatorAssetId
	responseData.Value = transaction.Value
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.TransactionStatus = transaction.TransactionStatus

	controller.Logger.Info("Outgoing response to InternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}
