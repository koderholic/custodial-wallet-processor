package controllers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
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
		asset := dto.Asset{}

		if err := controller.Repository.GetByFieldName(&dto.Asset{Symbol: assetSymbol, IsEnabled: true}, &asset); err != nil {
			controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", err)
			if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset record (%s) could not be created for user. %s", assetSymbol, err)))
				return
			}

			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", assetSymbol)))
			return
		}
		userAsset := dto.UserBalance{AssetID: asset.ID, UserID: requestData.UserID}
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

	if err := controller.Repository.GetAssetsByUserID(&dto.UserAssetBalance{UserID: userID}, &responseData); err != nil {
		controller.Logger.Error("Outgoing response to GetUserAssets request %+v", err)
		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", err.(utility.AppError).Error()))
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssets request %+v", responseData)

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}

// CreditUserAssets ... Credit a user asset abalance with the specified value
func (controller UserAssetController) CreditUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreditUserAssetRequest{}
	responseData := model.CreditUserAssetResponse{}
	transactionRef := utility.RandomString(16)

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("User asset account (%s) could not be credited. %s", requestData.Asset.AssetSymbol, err)))
		return
	}

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

	// ensure asset is supported
	asset := dto.Asset{}
	assetDetails := dto.UserBalance{}
	if err := controller.Repository.GetByFieldName(&dto.Asset{Symbol: requestData.Asset.AssetSymbol, IsEnabled: true}, &asset); err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("User asset account (%s) could not be credited. %s", requestData.Asset.AssetSymbol, err)))
			return
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", requestData.Asset.AssetSymbol)))
		return
	}
	// Get user asset account and ensure user has asset created for account
	if err := controller.Repository.GetByFieldName(&dto.UserBalance{UserID: requestData.UserID, Symbol: requestData.Asset.AssetSymbol}, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", err.(utility.AppError).Error()))
		return
	}

	// increment user account by volume
	var currentAvailableBalance, currentReservedBalance big.Float
	value, err := strconv.ParseFloat(requestData.Asset.Value, 32)
	creditValue := big.NewFloat(value)

	prevBal, err := strconv.ParseFloat(assetDetails.AvailableBalance, 32)
	curAvailableBalance := big.NewFloat(prevBal)

	cuBal, err := strconv.ParseFloat(assetDetails.ReservedBalance, 32)
	curreReservedBalance := big.NewFloat(cuBal)

	currentAvailableBalance.SetPrec(32)
	currentReservedBalance.SetPrec(32)
	currentAvailableBalance.Add(curAvailableBalance, creditValue)
	currentReservedBalance.Add(curreReservedBalance, creditValue)
	fmt.Printf("requestData > %+v > %+v > %+v", creditValue, curAvailableBalance, currentAvailableBalance.String())

	previousBalance := assetDetails.AvailableBalance

	if err != nil {

		fmt.Printf("responseData > %+v", err)
	}
	// currentAvailableBalance := assetDetails.AvailableBalance // + requestData.Asset.Value*math.Pow10(asset.Decimal)
	// currentReservedBalance := assetDetails.ReservedBalance   // + requestData.Asset.Value*math.Pow10(asset.Decimal)
	if err := controller.Repository.Db().Model(&dto.UserBalance{BaseDTO: dto.BaseDTO{ID: assetDetails.ID}}).Updates(dto.UserBalance{AvailableBalance: currentAvailableBalance.String(), ReservedBalance: currentReservedBalance.String()}).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", strings.Join(strings.Split(err.Error(), " ")[2:], " ")))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{
		Asset:                asset.Symbol,
		InitiatorID:          assetDetails.UserID,
		RecipientID:          assetDetails.UserID,
		TransactionReference: transactionRef,
		TransactionType:      dto.TransactionType.OFFCHAIN,
		TransactionStatus:    dto.TransactionStatus.COMPLETED,
		TransactionTag:       dto.TransactionTag.CREDIT,
		Value:                requestData.Asset.Value,
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance.String(),
		ReservedBalance:      currentReservedBalance.String(),
		ProcessingType:       dto.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	if err := tx.Commit().Error; err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	responseData.ID = transaction.ID
	responseData.Asset = transaction.Asset
	responseData.InitiatorID = transaction.InitiatorID
	responseData.RecipientID = transaction.RecipientID
	responseData.TransactionReference = transaction.TransactionReference
	responseData.TransactionType = transaction.TransactionType
	responseData.TransactionStatus = transaction.TransactionStatus
	responseData.TransactionTag = transaction.TransactionTag
	responseData.Value = transaction.Value
	responseData.PreviousBalance = previousBalance                   // / math.Pow10(asset.Decimal)
	responseData.AvailableBalance = currentAvailableBalance.String() // / math.Pow10(asset.Decimal)
	responseData.ReservedBalance = currentReservedBalance.String()   /// math.Pow10(asset.Decimal)
	responseData.ProcessingType = transaction.ProcessingType
	responseData.TransactionStartDate = transaction.TransactionStartDate
	responseData.TransactionEndDate = time.Now()
	responseData.CreatedAt = transaction.CreatedAt
	responseData.UpdatedAt = transaction.UpdatedAt

	controller.Logger.Info("Outgoing response to CreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}
