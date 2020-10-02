package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/services"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"
	Response "wallet-adapter/utility/response"

	"github.com/gorilla/mux"
)

// CreateUserAssets ... Creates all supported crypto asset record on the given user account
func (controller UserAssetController) CreateUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.CreateUserAssetRequest{}
	responseData := dto.UserAssetResponse{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("CreateUserAssets Logs : Incoming request details > %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// Create user asset record for each given denominationcontroller
	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)
	userAsset, err := UserAssetService.CreateAssets(requestData.Assets, requestData.UserID)
	if err != nil {
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
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
	apiResponse := Response.New()

	routeParams := mux.Vars(requestReader)
	userID, err := utility.ToUUID(routeParams["userId"])
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "GetUserAssets", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}
	logger.Info("GetUserAssets Logs : Incoming request details > userId : %+v", userID)

	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)
	userAsset, err := UserAssetService.FetchAssets(userID)
	if err != nil {
		ReturnError(responseWriter, "GetUserAssets", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
		return
	}

	responseData.Assets = userAsset
	logger.Info("GetUserAssets Logs : Outgoing response to request > %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssetById... Get user asset balance by id
func (controller UserAssetController) GetUserAssetById(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()

	routeParams := mux.Vars(requestReader)
	assetID, err := utility.ToUUID(routeParams["assetId"])
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "GetUserAssetById", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}
	logger.Info("GetUserAssetById Logs : Incoming request details > assetId : %+v", assetID)

	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)
	responseData, err := UserAssetService.GetAssetById(assetID)
	if err != nil {
		ReturnError(responseWriter, "GetUserAssetById", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetUserAssetByAddress ... Get user asset balance by address
func (controller UserAssetController) GetUserAssetByAddress(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	responseData := dto.Asset{}

	routeParams := mux.Vars(requestReader)
	address := routeParams["address"]
	assetSymbol := requestReader.URL.Query().Get("assetSymbol")
	userAssetMemo := requestReader.URL.Query().Get("userAssetMemo")
	logger.Info("Incoming request details for GetUserAssetByAddress : address : %+v, memo : %v, symbol : %s", address, userAssetMemo, assetSymbol)

	// Ensure assetSymbol is not empty
	if assetSymbol == "" {
		err := appError.Err{ErrCode: http.StatusBadRequest, ErrType: errorcode.INPUT_ERR_CODE, Err: errors.New(fmt.Sprintf("assetSymbol cannot be empty"))}
		ReturnError(responseWriter, "GetUserAssetByAddress", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}

	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)
	responseData, err := UserAssetService.GetAssetByAddressSymbolAndMemo(address, assetSymbol, userAssetMemo)
	if err != nil {
		ReturnError(responseWriter, "GetUserAssetByAddress", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) CreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.CreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for CreditUserAssets : %+v", requestData)
	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// Ensure asset exist and get asset
	assetDetails, err := UserAssetService.GetAssetBy(requestData.AssetID)
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.PlainError(err.ErrType, err.Error()))
	}
	// credit asset
	responseData, err = UserAssetService.CreditAsset(requestData, assetDetails, controller.GetInitiatingServiceId(requestReader))
	if err != nil {
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
		return
	}

	logger.Info("Outgoing response to CreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) OnChainCreditUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.OnChainCreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}
	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for OnChainCreditUserAssets : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// Ensure asset exist and get asset
	assetDetails, err := UserAssetService.GetAssetBy(requestData.AssetID)
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "OnChainCreditUserAsset", err, apiResponse.PlainError(err.ErrType, err.Error()))
	}
	// credit asset
	requestDetails := dto.CreditUserAssetRequest{AssetID: requestData.AssetID, Value: requestData.Value, TransactionReference: requestData.TransactionReference, Memo: requestData.Memo}
	responseData, err = UserAssetService.OnChainCreditAsset(requestDetails, requestData.ChainData, assetDetails, controller.GetInitiatingServiceId(requestReader))
	if err != nil {
		ReturnError(responseWriter, "OnChainCreditUserAsset", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
		return
	}

	logger.Info("Outgoing response to OnChainCreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// InternalTransfer ... transfer between two users
func (controller UserAssetController) InternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.InternalTransferRequest{}
	responseData := dto.TransactionReceipt{}
	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for InternalTransfer : %+v", requestData)
	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}
	// Ensure asset exist and get asset
	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)
	initiatorAssetDetails, err := UserAssetService.GetAssetBy(requestData.InitiatorAssetId)
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}
	// Ensure asset exist and get asset
	recipientAssetDetails, err := UserAssetService.GetAssetBy(requestData.RecipientAssetId)
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}
	// Ensure transfer cannot be done to self
	if requestData.InitiatorAssetId == requestData.RecipientAssetId {
		appErr := appError.Err{ErrType: errorcode.INPUT_ERR_CODE, ErrCode: http.StatusBadGateway, Err: errors.New(errorcode.NON_MATCHING_DENOMINATION)}
		ReturnError(responseWriter, "InternalTransfer", appErr, apiResponse.PlainError(appErr.ErrType, appErr.Error()))
		return
	}
	// Check if the denomination in the transction request is same for initiator and recipient
	if initiatorAssetDetails.DenominationID != recipientAssetDetails.DenominationID {
		appErr := appError.Err{ErrType: errorcode.INPUT_ERR_CODE, ErrCode: http.StatusBadGateway, Err: errors.New(errorcode.NON_MATCHING_DENOMINATION)}
		ReturnError(responseWriter, "InternalTransfer", appErr, apiResponse.PlainError(appErr.ErrType, appErr.Error()))
		return
	}
	// Checks if initiator has enough value to transfer
	if !utility.IsGreater(requestData.Value, initiatorAssetDetails.AvailableBalance, initiatorAssetDetails.Decimal) {
		appErr := appError.Err{ErrType: errorcode.INSUFFICIENT_FUNDS, ErrCode: http.StatusBadGateway, Err: errors.New(errorcode.INSUFFICIENT_FUNDS_ERR)}
		ReturnError(responseWriter, "InternalTransfer", appErr, apiResponse.PlainError(appErr.ErrType, appErr.Error()))
		return
	}
	// Call user asset service
	requestDetails := dto.CreditUserAssetRequest{Value: requestData.Value, TransactionReference: requestData.TransactionReference, Memo: requestData.Memo}
	responseData, err = UserAssetService.InternalTransfer(requestDetails, initiatorAssetDetails, recipientAssetDetails)
	if err != nil {
		ReturnError(responseWriter, "InternalTransfer", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
		return
	}

	logger.Info("Outgoing response to InternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// DebitUserAsset ... debit a user asset abalance with the specified value
func (controller UserAssetController) DebitUserAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.CreditUserAssetRequest{}
	responseData := dto.TransactionReceipt{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for DebitUserAsset : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// Ensure asset exist and get asset
	UserAssetService := services.NewUserAssetService(controller.Cache, controller.Config, controller.Repository)
	assetDetails, err := UserAssetService.GetAssetBy(requestData.AssetID)
	if err != nil {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError(appErr.ErrType, appErr.Error()))
	}

	// Checks if user asset has enough value to for the transaction
	if !utility.IsGreater(requestData.Value, assetDetails.AvailableBalance, assetDetails.Decimal) {
		err := appError.Err{ErrType: "INSUFFICIENT_FUNDS_ERR", ErrCode: http.StatusBadGateway, Err: errors.New(errorcode.INSUFFICIENT_FUNDS_ERR)}
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}

	responseData, err = UserAssetService.DebitAsset(requestData, assetDetails, controller.GetInitiatingServiceId(requestReader))
	if err != nil {
		ReturnError(responseWriter, "DebitUserAsset", err, apiResponse.PlainError(err.(appError.Err).ErrType, err.(appError.Err).Error()))
		return
	}

	logger.Info("Outgoing response to DebitUserAsset request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}
