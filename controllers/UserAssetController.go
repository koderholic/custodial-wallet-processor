package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"github.com/jinzhu/gorm"
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
	responseData := model.CreateUserAssetResponse{}

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

	// Create user asset record for each given asset
	for i := 0; i < len(requestData.Assets); i++ {
		assetSymbol := requestData.Assets[i]
		asset := dto.Denomination{}

		if err := controller.Repository.GetByFieldName(&dto.Denomination{Symbol: assetSymbol, IsEnabled: true}, &asset); err != nil {
			controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", err)
			if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset record (%s) could not be created for user. %s", assetSymbol, utility.GetSQLErr(err.(utility.AppError)))))
				return
			}

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", assetSymbol)))
			return
		}
		balance, _ := decimal.NewFromString("0.00")
		userAsset := dto.UserBalance{DenominationID: asset.ID, UserID: requestData.UserID, AvailableBalance:balance.String()}
		_ = controller.Repository.FindOrCreate(userAsset, &userAsset)
		userAsset.Symbol = asset.Symbol
		responseData.Assets = append(responseData.Assets, userAsset)
	}

	controller.Logger.Info("Outgoing response to CreateUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}

// GetUserAssets ... Get all user asset balance
func (controller UserAssetController) GetUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []dto.UserAssetBalance
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

	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{UserID: userID}, &responseData); err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))))
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssets request %+v", responseData)

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) DebitUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

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

	// ensure asset exists and then fetch asset
	assetDetails := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusNotFound)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// // increment user account by volume
	value, err := decimal.NewFromString(requestData.Value)
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
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestData.AssetID, err)))
		return
	}

	if err := tx.Model(&dto.UserBalance{BaseDTO: dto.BaseDTO{ID: assetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance - ?", value)).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to DebitUserAsset request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{
		Denomination:         assetDetails.Symbol,
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
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) CreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

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

	// ensure asset exists and fetc asset
	assetDetails := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusNotFound)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// // increment user account by volume
	value, err := decimal.NewFromString(requestData.Value)
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

	if err := tx.Model(&dto.UserBalance{BaseDTO: dto.BaseDTO{ID: assetDetails.ID}}).Updates(dto.UserBalance{AvailableBalance: currentAvailableBalance, }).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{
		Denomination:         assetDetails.Symbol,
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
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) InternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.InternalTransferRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

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

	// ensure asset exists and then fetch asset
	initiatorAssetDetails := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: requestData.InitiatorAssetId}}, &initiatorAssetDetails); err != nil {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusNotFound)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}
	recipientAssetDetails := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: requestData.RecipientAssetId}}, &recipientAssetDetails); err != nil {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusNotFound)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Ensure transfer cannot be done to self
	if requestData.InitiatorAssetId ==  requestData.RecipientAssetId {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", utility.NON_MATCHING_DENOMINATION)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.TRANSFER_TO_SELF))
		return
	}

	// Check if the denomnatio in transction is same for initiator and recipient
	if initiatorAssetDetails.DenominationID !=  recipientAssetDetails.DenominationID {
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", utility.NON_MATCHING_DENOMINATION)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.NON_MATCHING_DENOMINATION))
		return
	}

	// // increment user account by volume
	value, err := decimal.NewFromString(requestData.Value)
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
	if err := tx.Model(&dto.UserBalance{BaseDTO: dto.BaseDTO{ID: initiatorAssetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance - ?", value)).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}
	// Credit recipient
	if err := tx.Model(&dto.UserBalance{BaseDTO: dto.BaseDTO{ID: recipientAssetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance + ?", value)).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to InternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{
		Denomination:         initiatorAssetDetails.Symbol,
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
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}