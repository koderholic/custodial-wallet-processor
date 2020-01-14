package controllers

import (
	"encoding/json"
	"fmt"
	"math"
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
		asset := dto.Denomination{}

		if err := controller.Repository.GetByFieldName(&dto.Denomination{Symbol: assetSymbol, IsEnabled: true}, &asset); err != nil {
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
		userAsset := dto.UserBalance{DenominationID: asset.ID, UserID: requestData.UserID}
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
	paymentRef := utility.RandomString(16)

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestData.AssetID, err)))
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

	// ensure asset exists and fetc asset
	assetDetails := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: requestData.AssetID}}, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("User asset account (%s) could not be credited. %s", requestData.AssetID, err)))
			return
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) does not exist", requestData.AssetID)))
		return
	}

	// increment user account by volume
	value, err := strconv.ParseFloat(requestData.Value, 64)
	availBal, err := strconv.ParseFloat(assetDetails.AvailableBalance, 64)
	reserveBal, err := strconv.ParseFloat(assetDetails.ReservedBalance, 64)
	currentAvailableBalanceInFloat := availBal + value*math.Pow10(assetDetails.Decimal)
	currentReservedBalanceInFloat := reserveBal + value*math.Pow10(assetDetails.Decimal)
	previousBalance := strconv.FormatFloat(availBal, 'g', 1, 64)
	currentAvailableBalance := strconv.FormatFloat(currentAvailableBalanceInFloat, 'g', assetDetails.Decimal, 64)
	currentReservedBalance := strconv.FormatFloat(currentReservedBalanceInFloat, 'g', assetDetails.Decimal, 64)
	if err != nil {
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", strings.Join(strings.Split(err.Error(), " ")[2:], " ")))
		return
	}

	if err := tx.Model(&dto.UserBalance{BaseDTO: dto.BaseDTO{ID: assetDetails.ID}}).Updates(dto.UserBalance{AvailableBalance: currentAvailableBalance, ReservedBalance: currentReservedBalance}).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to CreditUserAssets request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", strings.Join(strings.Split(err.Error(), " ")[2:], " ")))
		return
	}

	// Create transaction record
	transaction := dto.Transaction{
		Denomination:         assetDetails.Symbol,
		InitiatorID:          assetDetails.ID, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestData.Memo,
		TransactionType:      dto.TransactionType.OFFCHAIN,
		TransactionStatus:    dto.TransactionStatus.COMPLETED,
		TransactionTag:       dto.TransactionTag.CREDIT,
		Value:                requestData.Value,
		PreviousBalance:      previousBalance,
		AvailableBalance:     currentAvailableBalance,
		ReservedBalance:      currentReservedBalance,
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
