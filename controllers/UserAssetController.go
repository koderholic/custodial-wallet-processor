package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

// CreateUserAssets ... Creates all supported crypto asset record on the given user account
func (controller UserAssetController) CreateUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.CreateUserAssetRequest{}
	responseData := dto.UserAssetResponse{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for CreateUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "CreateUserAssets", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	// Create user asset record for each given denomination
	for i := 0; i < len(requestData.Assets); i++ {
		denominationSymbol := requestData.Assets[i]
		denomination := model.Denomination{}

		if err := controller.Repository.GetByFieldName(&model.Denomination{AssetSymbol: denominationSymbol, IsEnabled: true}, &denomination); err != nil {
			if err.Error() == utility.SQL_404 {
				ReturnError(responseWriter, "CreateUserAssets", http.StatusNotFound, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", denominationSymbol)), controller.Logger)
				return
			}
			ReturnError(responseWriter, "CreateUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
			return
		}
		balance, _ := decimal.NewFromString("0.00")
		userAssetmodel := model.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID, AvailableBalance: balance.String()}
		_ = controller.Repository.FindOrCreateAssets(model.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID}, &userAssetmodel)

		userAsset := dto.Asset{}
		userAsset.ID = userAssetmodel.ID
		userAsset.UserID = userAssetmodel.UserID
		userAsset.AssetSymbol = userAssetmodel.AssetSymbol
		userAsset.AvailableBalance = userAssetmodel.AvailableBalance
		userAsset.Decimal = userAssetmodel.Decimal

		responseData.Assets = append(responseData.Assets, userAsset)
	}

	controller.Logger.Info("Outgoing response to CreateUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssets ... Get all user asset balance
func (controller UserAssetController) GetUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var userAssets []model.UserAsset
	responseData := dto.UserAssetResponse{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	userID, err := uuid.FromString(routeParams["userId"])
	if err != nil {
		ReturnError(responseWriter, "GetUserAssets", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetUserAssets : userID : %+v", userID)

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{UserID: userID}, &userAssets); err != nil {
		ReturnError(responseWriter, "GetUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get userAssets with userId = %s", utility.GetSQLErr(err.(utility.AppError)), userID)), controller.Logger)
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssets request %+v", userAssets)

	for i := 0; i < len(userAssets); i++ {
		userAsset := dto.Asset{}
		userAssetmodel := userAssets[i]

		userAsset.ID = userAssetmodel.ID
		userAsset.UserID = userAssetmodel.UserID
		userAsset.AssetSymbol = userAssetmodel.AssetSymbol
		userAsset.AvailableBalance = userAssetmodel.AvailableBalance
		userAsset.Decimal = userAssetmodel.Decimal

		responseData.Assets = append(responseData.Assets, userAsset)
	}
	if len(responseData.Assets) <= 0 {
		responseData.Assets = []dto.Asset{}
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssetById... Get user asset balance by id
func (controller UserAssetController) GetUserAssetById(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var userAssets model.UserAsset
	responseData := dto.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetUserAssetById", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetUserAssetById : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAssets); err != nil {
		ReturnError(responseWriter, "GetUserAssetById", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
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

	var userAsset model.UserAsset
	responseData := dto.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	address := routeParams["address"]
	assetSymbol := requestReader.URL.Query().Get("assetSymbol")
	userAssetMemo := requestReader.URL.Query().Get("userAssetMemo")

	controller.Logger.Info("Incoming request details for GetUserAssetByAddress : address : %+v", address)

	// Ensure assetSymbol is not empty
	if assetSymbol == "" {
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusBadRequest, "AssetSymbol cannot be empty", apiResponse.PlainError("INPUT_ERR", "AssetSymbol cannot be empty"), controller.Logger)
		return
	}

	// Check if asset is supported
	denomination := model.Denomination{}
	if err := controller.Repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		if err.Error() == utility.SQL_404 {
			ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusNotFound, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", assetSymbol)), controller.Logger)
			return
		}
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s, for get denomination with assetSymbol = %s", utility.GetSQLErr(err.(utility.AppError)), assetSymbol)), controller.Logger)
		return
	}

	// Ensure Memos are provided for v2_addresses
	IsV2Address, err := services.CheckV2Address(controller.Repository, address)
	if err != nil {
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR), controller.Logger)
		return
	}

	if IsV2Address {
		if userAssetMemo == "" {
			ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.EMPTY_MEMO_ERR), controller.Logger)
			return
		}
		userAsset, err = services.GetAssetForV2Address(controller.Repository, address, assetSymbol, userAssetMemo)
	} else {
		userAsset, err = services.GetAssetForV1Address(controller.Repository, address, assetSymbol)
	}

	if userAsset.AssetSymbol == "" {
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusNotFound, utility.SQL_404, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Record not found for asset address : %s, with asset symbol : %s and memo : %s", address, assetSymbol, userAssetMemo)), controller.Logger)
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssetByAddress request %+v", userAsset)

	responseData.ID = userAsset.ID
	responseData.UserID = userAsset.UserID
	responseData.AssetSymbol = userAsset.AssetSymbol
	responseData.AvailableBalance = userAsset.AvailableBalance
	responseData.Decimal = userAsset.Decimal

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) CreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.CreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for CreditUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "CreditUserAssets", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}
	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and fetc asset
	assetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "CreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get assetDetails with id = %s", utility.GetSQLErr(err), requestData.AssetID)), controller.Logger)
		return
	}

	// increment user account by value
	value := strconv.FormatFloat(requestData.Value, 'g', utility.DigPrecision, 64)
	currentAvailableBalance := utility.Add(requestData.Value, assetDetails.AvailableBalance, assetDetails.Decimal)

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "CreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)), controller.Logger)
		return
	}

	if err := tx.Model(assetDetails).Updates(model.UserAsset{AvailableBalance: currentAvailableBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "CreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		TransactionTag:       model.TransactionTag.CREDIT,
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
		ReturnError(responseWriter, "CreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "CreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)), controller.Logger)
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
	requestData := dto.OnChainCreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for OnChainCreditUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and fetc asset
	assetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get assetDetails with id = %s", utility.GetSQLErr(err), requestData.AssetID)), controller.Logger)
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
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)), controller.Logger)
		return
	}

	if err := tx.Model(&assetDetails).Updates(model.UserAsset{AvailableBalance: currentAvailableBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	//save chain tx model first, get id and use that in Transaction model
	var chainTransaction dto.ChainTransaction
	if err := tx.FirstOrCreate(&chainTransaction, dto.ChainTransaction{
		Status:          *requestData.ChainData.Status,
		TransactionHash: requestData.ChainData.TransactionHash,
		TransactionFee:  requestData.ChainData.TransactionFee,
		BlockHeight:     requestData.ChainData.BlockHeight,
	}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
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
	requestData := dto.InternalTransferRequest{}
	responseData := dto.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for InternalTransfer : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "InternalTransfer", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	initiatorAssetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.InitiatorAssetId}}, &initiatorAssetDetails); err != nil {
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get initiatorAssetDetails with id = %s", utility.GetSQLErr(err), requestData.InitiatorAssetId)), controller.Logger)
		return
	}
	recipientAssetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.RecipientAssetId}}, &recipientAssetDetails); err != nil {
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get initiatorAssetDetails with id = %s", utility.GetSQLErr(err), requestData.RecipientAssetId)), controller.Logger)
		return
	}

	// Ensure transfer cannot be done to self
	if requestData.InitiatorAssetId == requestData.RecipientAssetId {
		ReturnError(responseWriter, "InternalTransfer", http.StatusBadRequest, utility.NON_MATCHING_DENOMINATION, apiResponse.PlainError("INPUT_ERR", utility.TRANSFER_TO_SELF), controller.Logger)
		return
	}

	// Check if the denomination in the transction request is same for initiator and recipient
	if initiatorAssetDetails.DenominationID != recipientAssetDetails.DenominationID {
		ReturnError(responseWriter, "InternalTransfer", http.StatusBadRequest, utility.NON_MATCHING_DENOMINATION, apiResponse.PlainError("INPUT_ERR", utility.NON_MATCHING_DENOMINATION), controller.Logger)
		return
	}

	// Increment initiator asset balance and decrement recipient asset balance
	value := strconv.FormatFloat(requestData.Value, 'g', utility.DigPrecision, 64)
	initiatorCurrentBalance := utility.Subtract(requestData.Value, initiatorAssetDetails.AvailableBalance, initiatorAssetDetails.Decimal)
	recipientCurrentBalance := utility.Add(requestData.Value, recipientAssetDetails.AvailableBalance, recipientAssetDetails.Decimal)

	// Checks if initiator has enough value to transfer
	if !utility.IsGreater(requestData.Value, initiatorAssetDetails.AvailableBalance, initiatorAssetDetails.Decimal) {
		ReturnError(responseWriter, "InternalTransfer", http.StatusBadRequest, utility.INSUFFICIENT_FUNDS, apiResponse.PlainError("INPUT_ERR", utility.INSUFFICIENT_FUNDS), controller.Logger)
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR), controller.Logger)
		return
	}

	// Debit Inititor
	if err := tx.Model(&model.UserAsset{BaseModel: model.BaseModel{ID: initiatorAssetDetails.ID}}).Update(model.UserAsset{AvailableBalance: initiatorCurrentBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Credit recipient
	if err := tx.Model(&model.UserAsset{BaseModel: model.BaseModel{ID: recipientAssetDetails.ID}}).Update(model.UserAsset{AvailableBalance: recipientCurrentBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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

// DebitUserAsset ... debit a user asset abalance with the specified value
func (controller UserAssetController) DebitUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.CreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for DebitUserAsset : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "DebitUserAsset", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	assetDetails := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "DebitUserAsset", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get assetDetails with id = %s", utility.GetSQLErr(err), requestData.AssetID)), controller.Logger)
		return
	}

	// // decrement user account by value
	value := strconv.FormatFloat(requestData.Value, 'g', utility.DigPrecision, 64)
	currentAvailableBalance := utility.Subtract(requestData.Value, assetDetails.AvailableBalance, assetDetails.Decimal)

	// Checks if user asset has enough value to for the transaction
	if !utility.IsGreater(requestData.Value, assetDetails.AvailableBalance, assetDetails.Decimal) {
		ReturnError(responseWriter, "DebitUserAsset", http.StatusBadRequest, utility.INSUFFICIENT_FUNDS, apiResponse.PlainError("INSUFFICIENT_FUNDS", utility.INSUFFICIENT_FUNDS), controller.Logger)
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "DebitUserAsset", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestData.AssetID, err)), controller.Logger)
		return
	}
	if err := tx.Model(&model.UserAsset{BaseModel: model.BaseModel{ID: assetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance - ?", value)).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "DebitUserAsset", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		ReturnError(responseWriter, "DebitUserAsset", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "DebitUserAsset", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestData.AssetID, err)), controller.Logger)
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
