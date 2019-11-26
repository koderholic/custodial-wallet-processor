package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

func (c AssetController) GetAsset(w http.ResponseWriter, r *http.Request) {

	responseData := model.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(r)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("UUID_CAST_ERR", utility.UUID_CAST_ERR))
		return
	}

	c.Logger.Info("Incoming request details: %+v", assetID)

	if err := c.Repository.Get(assetID, &responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
		return
	}

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// FetchSupportedAssets ...
func (c AssetController) FetchSupportedAssets(w http.ResponseWriter, r *http.Request) {

	var responseData []model.Asset
	apiResponse := utility.NewResponse()

	if err := c.Repository.GetSupportedCrypto(&responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
		return
	}

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// FetchAllAssets ...
func (c AssetController) FetchAllAssets(w http.ResponseWriter, r *http.Request) {

	var responseData []model.Asset
	apiResponse := utility.NewResponse()

	if err := c.Repository.Fetch(&responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
		return
	}

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}
