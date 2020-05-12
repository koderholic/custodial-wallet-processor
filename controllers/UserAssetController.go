package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"

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
		ReturnError(responseWriter, "CreateUserAssets", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	// Create user asset record for each given denomination
	for i := 0; i < len(requestData.Assets); i++ {
		denominationSymbol := requestData.Assets[i]
		denomination := dto.Denomination{}

		if err := controller.Repository.GetByFieldName(&dto.Denomination{AssetSymbol: denominationSymbol, IsEnabled: true}, &denomination); err != nil {
			if err.Error() == utility.SQL_404 {
				ReturnError(responseWriter, "CreateUserAssets", http.StatusNotFound, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", denominationSymbol)), controller.Logger)
				return
			}
			ReturnError(responseWriter, "CreateUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
			return
		}
		balance, _ := decimal.NewFromString("0.00")
		userAssetDTO := dto.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID, AvailableBalance: balance.String()}
		_ = controller.Repository.FindOrCreateAssets(dto.UserAsset{DenominationID: denomination.ID, UserID: requestData.UserID}, &userAssetDTO)

		userAsset := model.Asset{}
		userAsset.ID = userAssetDTO.ID
		userAsset.UserID = userAssetDTO.UserID
		userAsset.AssetSymbol = userAssetDTO.AssetSymbol
		userAsset.AvailableBalance = userAssetDTO.AvailableBalance
		userAsset.Decimal = userAssetDTO.Decimal

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
		ReturnError(responseWriter, "GetUserAssets", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetUserAssets : userID : %+v", userID)

	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{UserID: userID}, &userAssets); err != nil {
		ReturnError(responseWriter, "GetUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
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
		ReturnError(responseWriter, "GetUserAssetById", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetUserAssetById : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetID}}, &userAssets); err != nil {
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

	var userAsset dto.UserAsset
	var userAddresses []dto.UserAddress
	responseData := model.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	address := routeParams["address"]
	assetSymbol := requestReader.URL.Query().Get("assetSymbol")

	controller.Logger.Info("Incoming request details for GetUserAssetByAddress : address : %+v", address)

	// Ensure assetSymbol is not empty
	if assetSymbol == "" {
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusBadRequest, "AssetSymbol cannot be empty", apiResponse.PlainError("INPUT_ERR", "AssetSymbol cannot be empty"), controller.Logger)
		return
	}

	// Check if asset is supported
	denomination := dto.Denomination{}
	if err := controller.Repository.GetByFieldName(&dto.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		if err.Error() == utility.SQL_404 {
			ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusNotFound, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", assetSymbol)), controller.Logger)
			return
		}
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
		return
	}

	if err := controller.Repository.FetchByFieldName(&dto.UserAddress{Address: address}, &userAddresses); err != nil {
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
		return
	}
	for _, userAddress := range userAddresses {
		asset := dto.UserAsset{}
		if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: userAddress.AssetID}}, &asset); err != nil {
			continue
		}
		if asset.AssetSymbol == assetSymbol {
			userAsset = asset
			break
		}
	}

	if userAsset.AssetSymbol == "" {
		ReturnError(responseWriter, "GetUserAssetByAddress", http.StatusNotFound, utility.SQL_404, apiResponse.PlainError("INPUT_ERR", utility.SQL_404), controller.Logger)
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
	requestData := model.CreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for CreditUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "CreditUserAssets", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}
	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and fetc asset
	assetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "CreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
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

	if err := tx.Model(assetDetails).Updates(dto.UserAsset{AvailableBalance: currentAvailableBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "CreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		Value:                value,
		PreviousBalance:      assetDetails.AvailableBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
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
	requestData := model.OnChainCreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for OnChainCreditUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and fetc asset
	assetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
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

	if err := tx.Model(&assetDetails).Updates(dto.UserAsset{AvailableBalance: currentAvailableBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		ReturnError(responseWriter, "OnChainCreditUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		Value:                value,
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
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
	requestData := model.InternalTransferRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for InternalTransfer : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "InternalTransfer", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	initiatorAssetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.InitiatorAssetId}}, &initiatorAssetDetails); err != nil {
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	recipientAssetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.RecipientAssetId}}, &recipientAssetDetails); err != nil {
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
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
	if err := tx.Model(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: initiatorAssetDetails.ID}}).Update(dto.UserAsset{AvailableBalance: initiatorCurrentBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Credit recipient
	if err := tx.Model(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: recipientAssetDetails.ID}}).Update(dto.UserAsset{AvailableBalance: recipientCurrentBalance}).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "InternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		Value:                value,
		PreviousBalance:      initiatorAssetDetails.AvailableBalance,
		AvailableBalance:     initiatorCurrentBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
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
	requestData := model.CreditUserAssetRequest{}
	responseData := model.TransactionReceipt{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for DebitUserAsset : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "DebitUserAsset", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// ensure asset exists and then fetch asset
	assetDetails := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		ReturnError(responseWriter, "DebitUserAsset", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
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
	if err := tx.Model(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetDetails.ID}}).Update("available_balance", gorm.Expr("available_balance - ?", value)).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "DebitUserAsset", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
		Value:                value,
		PreviousBalance:      assetDetails.AvailableBalance,
		AvailableBalance:     currentAvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
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
