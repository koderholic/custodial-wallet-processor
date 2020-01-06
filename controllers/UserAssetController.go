package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// GetUserAssets ... Get all user asset balance
func (controller UserAssetController) CreditUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreditUserAssetRequest{}
	responseData := model.CreditUserAssetResponse{}
	transactionRef := utility.

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
		controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", err)
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
		controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", err)
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

	// Create transaction record
	transaction := dto.Transaction{
		AssetID   : asset.ID
		InitiatorID   : "Bundle"
		Recipient   : assetDetails.UserID
		TransactionReference string       `gorm:"not null;" json:"transaction_reference,omitempty"`
		TransactionType      string       `gorm:"not null;default:'Offchain'" json:"transaction_type,omitempty"`
		TransactionStatus    string       `gorm:"not null;default:'Pending';index:transaction_status" json:"transaction_status,omitempty"`
		TransactionTag       string       `gorm:"not null;default:'Sell'" json:"transaction_tag,omitempty"`
		Volume               string       `gorm:"not null;default:'Sell'" json:"volume,omitempty"`
		AvailableBalance     float64      `gorm:"type:BIGINT;not null" json:"available_balance,omitempty"`
		ReservedBalance      float64      `gorm:"type:BIGINT;not null" json:"reserved_balance,omitempty"`
		ProcessingType       string       `gorm:"not null;default:'Single'" json:"processing_type,omitempty"`
		BatchID              uuid.UUID    `gorm:"type:VARCHAR(36);" json:"batch_id,omitempty"`
		TransactionStartDate time.Time    `json:"transaction_start_date,omitempty"`
		TransactionEndDate   time.Time    `json:"transaction_end_date,omitempty"`
		Batch                BatchRequest `sql:"-" json:"omitempty"`
	}
	if err := controller.Repository.Create(&assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", err)
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
	assetDetails.AvailableBalance = assetDetails.AvailableBalance + requestData.Asset.Volume
	assetDetails.ReservedBalance = assetDetails.ReservedBalance + requestData.Asset.Volume
	if err := controller.Repository.Update(assetDetails.ID, &assetDetails); err != nil {
		controller.Logger.Error("Outgoing response to CreateUserAssets request %+v", err)
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

	controller.Logger.Info("Outgoing response to CreditUserAssets request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}
