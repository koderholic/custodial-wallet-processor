package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// CreateUserAssets ... Creates all supported crypto asset record on the given user account
func (c UserAssetController) CreateUserAssets(w http.ResponseWriter, r *http.Request) {
	apiResponse := utility.NewResponse()

	var responseData []model.UserAssetBalance
	var supportedAssets []model.Asset

	routeParams := mux.Vars(r)
	userID, err := uuid.FromString(routeParams["userId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.UUIDCASTERROR, utility.GetCodeMsg(utility.UUIDCASTERROR)))
		return
	}

	if err := c.Repository.Fetch(&supportedAssets); err != nil {
		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, utility.GetCodeMsg(utility.SYSTEMERROR)))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, err.(utility.AppError).Error()))
		return
	}
	for i := 0; i < len(supportedAssets); i++ {
		assetID := supportedAssets[i].ID
		userAsset := model.UserBalance{AssetID: assetID, UserID: userID}
		if err := c.Repository.FindOrCreateUserAsset(userAsset, &userAsset); err != nil {
			if err.(utility.AppError).Type() == utility.SYSTEMERROR {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, utility.GetCodeMsg(utility.SYSTEMERROR)))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, err.(utility.AppError).Error()))
			return
		}
	}

	c.Repository.GetAssetsByUserID(userID, &responseData)
	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success(utility.SUCCESS, utility.GetCodeMsg(utility.SUCCESS), responseData))

}

// GetUserAssets ... Get all user asset balance
func (c UserAssetController) GetUserAssets(w http.ResponseWriter, r *http.Request) {

	var responseData []model.UserAssetBalance
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(r)
	userID, err := uuid.FromString(routeParams["userId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.INPUTERROR, "Cannot cast Id"))
		return
	}

	if err := c.Repository.GetAssetsByUserID(userID, &responseData); err != nil {
		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, utility.GetCodeMsg(utility.SYSTEMERROR)))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, err.(utility.AppError).Error()))
		return
	}
	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success(utility.SUCCESS, utility.GetCodeMsg(utility.SUCCESS), responseData))

}
