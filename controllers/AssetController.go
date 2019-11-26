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
	assetId, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("UUIDCASTERROR", utility.UUIDCASTERROR))
		return
	}

	c.Logger.Info("Incoming request details: %+v", assetId)

	if err := c.Repository.Get(assetId, &responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEMERROR", utility.SYSTEMERROR))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEMERROR", err.(utility.AppError).Error()))
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

		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEMERROR", utility.SYSTEMERROR))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEMERROR", err.(utility.AppError).Error()))
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

		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEMERROR", utility.SYSTEMERROR))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError("SYSTEMERROR", err.(utility.AppError).Error()))
		return
	}

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}
