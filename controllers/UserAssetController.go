package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"

	"github.com/jinzhu/gorm"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// CreateUserAssets ... Creates all supported crypto asset record on the given user account
func (controller UserAssetController) CreateUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.CreateUserAssetRequest{}
	responseData := dto.UserAssetResponse{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("CreateUserAssets Logs : Incoming request details > %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(utility.AppError).ErrData.([]map[string]string)) > 0 {
		appErr := err.(utility.AppError)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// Create user asset record for each given denominationcontroller
	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config)
	userAsset, err := UserAssetService.CreateAsset(controller.Repository, requestData.Assets, requestData.UserID)
	if err != nil {
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.PlainError(err.(utility.AppError).ErrType, err.(utility.AppError).Error()))
		return
	}
	responseData.Assets = userAsset

	logger.Info("CreateUserAssets Logs : Outgoing response to request > %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssets ... Get all user asset balance
func (controller UserAssetController) GetUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	responseData := dto.UserAssetResponse{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	userID, err := utility.ToUUID(routeParams["userId"])
	if err != nil {
		err := err.(utility.AppError)
		ReturnError(responseWriter, "GetUserAssets", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}
	logger.Info("GetUserAssets Logs : Incoming request details > userId : %+v", userID)

	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config)
	userAsset, err := UserAssetService.FetchAssets(controller.Repository, userID)
	if err != nil {
		ReturnError(responseWriter, "GetUserAssets", err, apiResponse.PlainError(err.(utility.AppError).ErrType, err.(utility.AppError).Error()))
		return
	}
	responseData.Assets = userAsset
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssetById... Get user asset balance by id
func (controller UserAssetController) GetUserAssetById(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := utility.ToUUID(routeParams["assetId"])
	if err != nil {
		err := err.(utility.AppError)
		ReturnError(responseWriter, "GetUserAssetById", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}
	logger.Info("GetUserAssetById Logs : Incoming request details > assetId : %+v", assetID)

	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config)
	responseData, err := UserAssetService.GetAssetById(controller.Repository, assetID)
	if err != nil {
		ReturnError(responseWriter, "GetUserAssetById", err, apiResponse.PlainError(err.(utility.AppError).ErrType, err.(utility.AppError).Error()))
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssetByAddress ... Get user asset balance by address
func (controller UserAssetController) GetUserAssetByAddress(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	responseData := dto.Asset{}

	routeParams := mux.Vars(requestReader)
	address := routeParams["address"]
	assetSymbol := requestReader.URL.Query().Get("assetSymbol")
	userAssetMemo := requestReader.URL.Query().Get("userAssetMemo")
	logger.Info("Incoming request details for GetUserAssetByAddress : address : %+v, memo : %v, symbol : %s", address, userAssetMemo, assetSymbol)

	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config)
	responseData, err := UserAssetService.GetAssetByAddressSymbolAndMemo(controller.Repository, address, assetSymbol, userAssetMemo)
	if err != nil {
		ReturnError(responseWriter, "GetUserAssetByAddress", err, apiResponse.PlainError(err.(utility.AppError).ErrType, err.(utility.AppError).Error()))
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) CreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.CreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for CreditUserAssets : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(utility.AppError).ErrData.([]map[string]string)) > 0 {
		appErr := err.(utility.AppError)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// credit asset

	logger.Info("Outgoing response to CreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) OnChainCreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.OnChainCreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for OnChainCreditUserAssets : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(utility.AppError).ErrData.([]map[string]string)) > 0 {
		appErr := err.(utility.AppError)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}
	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and fetc asset
	assetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "OnChainCreditUserAssets", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, fmt.Sprintf("%s, for get assetDetails with id = %s", utility.GetSQLErr(err), requestData.AssetID)))
		return
	}

	// // increment user account by value
	value := strconv.FormatFloat(requestData.Value, 'g', utility.DigPrecision, 64)
	previousBalance := assetDetails.AvailableBalance
	currentAvailableBalance := utility.Add(requestData.Value, assetDetails.AvailableBalance, assetDetails.Decimal)

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "OnChainCreditUserAssets", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)))
		return
	}

	if err := tx.Model(&assetDetails).Updates(model.UserAsset{AvailableBalance: currentAvailableBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "OnChainCreditUserAssets", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}

	//save chain tx model first, get id and use that in Transaction model
	var chainTransaction model.ChainTransaction
	newChainTransaction := model.ChainTransaction{
		Status:           *requestData.ChainData.Status,
		TransactionHash:  requestData.ChainData.TransactionHash,
		TransactionFee:   requestData.ChainData.TransactionFee,
		BlockHeight:      requestData.ChainData.BlockHeight,
		RecipientAddress: requestData.ChainData.RecipientAddress,
	}
	if err := tx.Where(model.ChainTransaction{
		TransactionHash:  requestData.ChainData.TransactionHash,
		RecipientAddress: requestData.ChainData.RecipientAddress,
	}).Assign(newChainTransaction).FirstOrCreate(&chainTransaction).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "OnChainCreditUserAssets", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}

	transactionStatus := model.TransactionStatus.PENDING
	if chainTransaction.Status == true {
		transactionStatus = model.TransactionStatus.COMPLETED
	} else {
		transactionStatus = model.TransactionStatus.REJECTED
	}
	// Create transaction record
	transaction := model.Transaction{

		InitiatorID:          decodedToken.ServiceID, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      model.TransactionType.ONCHAIN,
		TransactionStatus:    transactionStatus,
		TransactionTag:       model.TransactionTag.DEPOSIT,
		Value:                value,
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       model.ProcessingType.SINGLE,
		OnChainTxId:          chainTransaction.ID,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
		AssetSymbol:          assetDetails.AssetSymbol,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "OnChainCreditUserAssets", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "OnChainCreditUserAssets", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, utility.GetSQLErr(err)))
		return
	}

	responseData.AssetID = requestData.AssetID
	responseData.Value = transaction.Value
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.TransactionStatus = transaction.TransactionStatus

	logger.Info("Outgoing response to OnChainCreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// InternalTransfer ... transfer between two users
func (controller UserAssetController) InternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.InternalTransferRequest{}
	responseData := dto.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for InternalTransfer : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(utility.AppError).ErrData.([]map[string]string)) > 0 {
		appErr := err.(utility.AppError)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	initiatorAssetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.InitiatorAssetId}}, &initiatorAssetDetails); err != nil {
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, fmt.Sprintf("%s, for get initiatorAssetDetails with id = %s", utility.GetSQLErr(err), requestData.InitiatorAssetId)))
		return
	}
	recipientAssetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.RecipientAssetId}}, &recipientAssetDetails); err != nil {
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, fmt.Sprintf("%s, for get initiatorAssetDetails with id = %s", utility.GetSQLErr(err), requestData.RecipientAssetId)))
		return
	}

	// Ensure transfer cannot be done to self
	if requestData.InitiatorAssetId == requestData.RecipientAssetId {
		ReturnError(responseWriter, "InternalTransfer", errorcode.NON_MATCHING_DENOMINATION, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, errorcode.TRANSFER_TO_SELF))
		return
	}

	// Check if the denomination in the transction request is same for initiator and recipient
	if initiatorAssetDetails.DenominationID != recipientAssetDetails.DenominationID {
		ReturnError(responseWriter, "InternalTransfer", errorcode.NON_MATCHING_DENOMINATION, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, errorcode.NON_MATCHING_DENOMINATION))
		return
	}

	// Increment initiator asset balance and decrement recipient asset balance
	value := strconv.FormatFloat(requestData.Value, 'g', utility.DigPrecision, 64)
	initiatorCurrentBalance := utility.Subtract(requestData.Value, initiatorAssetDetails.AvailableBalance, initiatorAssetDetails.Decimal)
	recipientCurrentBalance := utility.Add(requestData.Value, recipientAssetDetails.AvailableBalance, recipientAssetDetails.Decimal)

	// Checks if initiator has enough value to transfer
	if !utility.IsGreater(requestData.Value, initiatorAssetDetails.AvailableBalance, initiatorAssetDetails.Decimal) {
		ReturnError(responseWriter, "InternalTransfer", errorcode.INSUFFICIENT_FUNDS_ERR, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, errorcode.INSUFFICIENT_FUNDS_ERR))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError("SERVER_ERR", errorcode.SERVER_ERR))
		return
	}

	// Debit Inititor
	if err := tx.Model(&model.UserAsset{BaseModel: model.BaseModel{ID: initiatorAssetDetails.ID}}).Update(model.UserAsset{AvailableBalance: initiatorCurrentBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}

	// Credit recipient
	if err := tx.Model(&model.UserAsset{BaseModel: model.BaseModel{ID: recipientAssetDetails.ID}}).Update(model.UserAsset{AvailableBalance: recipientCurrentBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}

	// Create transaction record
	transaction := model.Transaction{
		InitiatorID:          initiatorAssetDetails.ID,
		RecipientID:          recipientAssetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      model.TransactionType.OFFCHAIN,
		TransactionStatus:    model.TransactionStatus.COMPLETED,
		TransactionTag:       model.TransactionTag.TRANSFER,
		Value:                value,
		PreviousBalance:      initiatorAssetDetails.AvailableBalance,
		AvailableBalance:     initiatorCurrentBalance,
		ProcessingType:       model.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
		AssetSymbol:          initiatorAssetDetails.AssetSymbol,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}

	responseData.AssetID = requestData.InitiatorAssetId
	responseData.Value = transaction.Value
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.TransactionStatus = transaction.TransactionStatus

	logger.Info("Outgoing response to InternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// DebitUserAsset ... debit a user asset abalance with the specified value
func (controller UserAssetController) DebitUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.CreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for DebitUserAsset : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(utility.AppError).ErrData.([]map[string]string)) > 0 {
		appErr := err.(utility.AppError)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	assetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, fmt.Sprintf("%s, for get assetDetails with id = %s", utility.GetSQLErr(err), requestData.AssetID)))
		return
	}

	// // decrement user account by value
	value := strconv.FormatFloat(requestData.Value, 'g', utility.DigPrecision, 64)
	currentAvailableBalance := utility.Subtract(requestData.Value, assetDetails.AvailableBalance, assetDetails.Decimal)

	// Checks if user asset has enough value to for the transaction
	if !utility.IsGreater(requestData.Value, assetDetails.AvailableBalance, assetDetails.Decimal) {
		ReturnError(responseWriter, "DebitUserAsset", errorcode.INSUFFICIENT_FUNDS_ERR, apiResponse.PlainError("INSUFFICIENT_FUNDS_ERR", errorcode.INSUFFICIENT_FUNDS_ERR))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestData.AssetID, err)))
		return
	}
	if err := tx.Model(&model.UserAsset{BaseModel: model.BaseModel{ID: assetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance - ?", value)).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}
	// Create transaction record
	transaction := model.Transaction{

		InitiatorID:          decodedToken.ServiceID, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      model.TransactionType.OFFCHAIN,
		TransactionStatus:    model.TransactionStatus.COMPLETED,
		TransactionTag:       model.TransactionTag.DEBIT,
		Value:                value,
		PreviousBalance:      assetDetails.AvailableBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       model.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
		AssetSymbol:          assetDetails.AssetSymbol,
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError("SERVER_ERR", utility.GetSQLErr(err)))
		return
	}
	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestData.AssetID, err)))
		return
	}
	responseData.AssetID = requestData.AssetID
	responseData.Value = transaction.Value
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.TransactionStatus = transaction.TransactionStatus

	logger.Info("Outgoing response to DebitUserAsset request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetTransaction ... Retrieves the transaction details of the reference sent
func (controller UserAssetController) GetTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData dto.TransactionResponse
	var transaction model.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	transactionRef := routeParams["reference"]
	logger.Info("Incoming request details for GetTransaction : transaction reference : %+v", transactionRef)

	if err := controller.Repository.GetByFieldName(&model.Transaction{TransactionReference: transactionRef}, &transaction); err != nil {
		logger.Error("Outgoing response to GetTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == errorcode.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError(errorcode.INPUT_ERR_CODE, fmt.Sprintf("%s, for get transaction with transactionReference = %s", utility.GetSQLErr(err), transactionRef)))
		return
	}
	if transaction.TransactionStatus == model.TransactionStatus.PROCESSING && transaction.TransactionType == model.TransactionType.ONCHAIN {
		status, _ := controller.verifyTransactionStatus(transaction)
		if status != "" {
			transaction.TransactionStatus = status
		}
	}

	transaction.Map(&responseData)
	controller.populateChainData(transaction, &responseData, apiResponse, responseWriter)
	logger.Info("Outgoing response to GetTransaction request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetTransactionsByAssetId ... Retrieves all transactions relating to an asset
func (controller UserAssetController) GetTransactionsByAssetId(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData dto.TransactionListResponse
	var initiatorTransactions []model.Transaction
	var recipientTransactions []model.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, errorcode.UUID_CAST_ERR))
		return
	}
	logger.Info("Incoming request details for GetTransactionsByAssetId : assetID : %+v", assetID)
	if err := controller.Repository.FetchByFieldName(&model.Transaction{InitiatorID: assetID}, &initiatorTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, utility.GetSQLErr(err)))
		return
	}
	if err := controller.Repository.FetchByFieldName(&model.Transaction{RecipientID: assetID}, &recipientTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, utility.GetSQLErr(err)))
		return
	}

	for i := 0; i < len(initiatorTransactions); i++ {
		transaction := initiatorTransactions[i]
		tx := dto.TransactionResponse{}
		transaction.Map(&tx)
		controller.populateChainData(transaction, &tx, apiResponse, responseWriter)
		responseData.Transactions = append(responseData.Transactions, tx)
	}
	for i := 0; i < len(recipientTransactions); i++ {
		receipientTransaction := recipientTransactions[i]
		txRecipient := dto.TransactionResponse{}
		receipientTransaction.Map(&txRecipient)
		controller.populateChainData(receipientTransaction, &txRecipient, apiResponse, responseWriter)
		responseData.Transactions = append(responseData.Transactions, txRecipient)
	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []dto.TransactionResponse{}
	}

	logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

func (controller UserAssetController) populateChainData(transaction model.Transaction, txResponse *dto.TransactionResponse, apiResponse utility.ResponseResultObj, responseWriter http.ResponseWriter) {
	//get and populate chain transaction if exists, if this call fails, log error but proceed on
	chainTransaction := model.ChainTransaction{}
	chainData := dto.ChainData{}
	if transaction.TransactionType == "ONCHAIN" && transaction.OnChainTxId != uuid.Nil {
		err := controller.Repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: transaction.OnChainTxId}}, &chainTransaction)
		if err != nil {
			ReturnError(responseWriter, "GetTransaction", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, utility.GetSQLErr(err)))
			txResponse.ChainData = nil
		} else {
			chainTransaction.MaptoDto(&chainData)
			txResponse.ChainData = &chainData
		}
	} else {
		txResponse.ChainData = nil
	}

}

func (controller UserAssetController) verifyTransactionStatus(transaction model.Transaction) (string, error) {

	// Get queued transaction for transactionId
	var transactionQueue model.TransactionQueue
	if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionId: transaction.ID}, &transactionQueue); err != nil {
		logger.Error("verifyTransactionStatus logs : Error while fetching corresponding queued transaction for transaction (%v) : %s", transaction.ID, err)
		return "", err
	}

	broadcastTXRef := transactionQueue.DebitReference
	serviceErr := dto.ExternalServicesRequestErr{}

	// Check if the transaction belongs to a batch and return batch
	BatchService := services.NewBatchService(controller.Cache, controller.Config)
	batchExist, _, err := BatchService.CheckBatchExistAndReturn(controller.Repository, transactionQueue.BatchID)
	if err != nil {
		logger.Error("verifyTransactionStatus logs :Error occured while checking if transaction is batched : %s", err)
		return "", err
	}
	if batchExist {
		broadcastTXRef = transactionQueue.BatchID.String()
	}

	// Get status of the TXN
	CryptoAdapterService := services.NewCryptoAdapterService(controller.Cache, controller.Config)
	txnExist, broadcastedTX, err := CryptoAdapterService.GetBroadcastedTXNDetailsByRef(broadcastTXRef, transactionQueue.AssetSymbol, controller.Cache, controller.Config)
	if err != nil {
		logger.Error("verifyTransactionStatus logs : Error checking the broadcasted state for queued transaction (%+v) : %s", transactionQueue.ID, err)
		return "", err
	}

	if !txnExist {
		if utility.IsExceedWaitTime(time.Since(transactionQueue.CreatedAt), time.Duration(utility.MIN_WAIT_TIME_IN_PROCESSING)) {
			// Revert the transaction status back to pending, as transaction has not been broadcasted
			if err := controller.updateTransactions(transactionQueue, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
				logger.Error("verifyTransactionStatus logs :Error occured while updating transaction %+v to PENDING : %+v; %s", transactionQueue.TransactionId, serviceErr, err)
				return "", err
			}
			return model.TransactionStatus.PENDING, err
		}
		return "", err
	}

	// Get the chain transaction for the broadcasted txn hash
	chainTransaction := model.ChainTransaction{}
	err = controller.Repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: transaction.OnChainTxId}}, &chainTransaction)
	if err != nil {
		logger.Error("verifyTransactionStatus logs : Error fetching chain transaction for transaction (%+v) : %s", transactionQueue.ID, err)
		return "", err
	}
	blockHeight, err := strconv.Atoi(broadcastedTX.BlockHeight)

	// Update the transactions on the transaction table and on queue tied to the chain transaction as well as the batch status,if it is a batch transaction
	switch broadcastedTX.Status {
	case utility.SUCCESSFUL:
		chainTransactionUpdate := model.ChainTransaction{Status: true, TransactionFee: broadcastedTX.TransactionFee, BlockHeight: int64(blockHeight)}
		if err := controller.Repository.Update(&chainTransaction, chainTransactionUpdate); err != nil {
			logger.Error("verifyTransactionStatus logs : Error updating chain transaction for transaction (%+v) : %s", transactionQueue.ID, err)
			return "", err
		}
		if err := controller.updateTransactions(transactionQueue, model.TransactionStatus.COMPLETED, chainTransaction); err != nil {
			logger.Error("verifyTransactionStatus logs : Error updating transaction (%+v) to COMPLETED : %s", transactionQueue.ID, err)
			return "", err
		}
		return model.TransactionStatus.COMPLETED, err
	case utility.FAILED:
		if err := controller.updateTransactions(transactionQueue, model.TransactionStatus.TERMINATED, chainTransaction); err != nil {
			logger.Error("verifyTransactionStatus logs : Error updating transaction (%+v) to TERMINTATED : %s", transactionQueue.ID, err)
			return "", err
		}
		return model.TransactionStatus.TERMINATED, err
	}

	return "", nil
}

func (controller UserAssetController) updateTransactions(transaction model.TransactionQueue, status string, chainTransaction model.ChainTransaction) error {

	BatchService := services.NewBatchService(controller.Cache, controller.Config)
	batchExist, batch, err := BatchService.CheckBatchExistAndReturn(controller.Repository, transaction.BatchID)
	if err != nil {
		return err
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		logger.Error("Error response from updateTransactions : %+v while creating db transaction", err)
		return err
	}

	if batchExist {
		if err := tx.Model(&model.Transaction{}).Where("batch_id = ?", transaction.BatchID).Updates(model.Transaction{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating transactions with batchId : %+v", err, transaction.BatchID)
			return err
		}
		if err := tx.Model(&model.TransactionQueue{}).Where("batch_id = ?", transaction.BatchID).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating queued transactions with batchId  : %+v", err, transaction.ID)
			return err
		}
		dateCompleted := time.Now()
		if err := tx.Model(&batch).Updates(model.BatchRequest{Status: status, DateCompleted: &dateCompleted}).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Model(&model.Transaction{}).Where("id = ?", transaction.TransactionId).Updates(model.Transaction{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating transaction with id : %+v", err, transaction.TransactionId)
			return err
		}
		if err := tx.Model(&model.TransactionQueue{}).Where("id = ?", transaction.ID).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating queued transaction with id  : %v", err, transaction.ID)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Error response from updateTransactions : %+v while commiting db transaction", err)
		return err
	}
	return nil

}
