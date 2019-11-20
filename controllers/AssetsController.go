package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

func (c Controller) AddSupportedAsset(w http.ResponseWriter, r *http.Request) {

	requestData := model.Asset{}
	apiResponse := utility.NewResponse()

	json.NewDecoder(r.Body).Decode(&requestData)
	c.Logger.Info("Incoming request details: %+v", requestData)

	if err := c.Repository.Create(&requestData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, utility.GetCodeMsg(utility.SYSTEMERROR)))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.Error(utility.SYSTEMERROR, err.(utility.AppError).Error(), requestData))
		return
	}

	responseData := requestData
	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success(utility.SUCCESS, utility.GetCodeMsg(utility.SUCCESS), responseData))

}

func (c Controller) UpdateAsset(w http.ResponseWriter, r *http.Request) {

	requestData := model.Asset{}
	apiResponse := utility.NewResponse()
	routeParams := mux.Vars(r)
	assetId, err := uuid.FromString(routeParams["assetId"])

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.INPUTERROR, "Cannot cast Id"))
		return
	}
	requestData.ID = assetId

	json.NewDecoder(r.Body).Decode(&requestData)
	c.Logger.Info("Incoming request details: %+v", requestData)

	if err := c.Repository.Update(assetId, &requestData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, utility.GetCodeMsg(utility.SYSTEMERROR)))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.Error(utility.SYSTEMERROR, err.(utility.AppError).Error(), requestData))
		return
	}

	responseData := requestData
	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.Success(utility.SUCCESS, utility.GetCodeMsg(utility.SUCCESS), responseData))

}

func (c Controller) GetAsset(w http.ResponseWriter, r *http.Request) {

	responseData := model.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(r)
	assetId, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.INPUTERROR, "Cannot cast Id"))
		return
	}

	c.Logger.Info("Incoming request details: %+v", assetId)
	fmt.Printf("responseData . %+v", responseData)

	if err := c.Repository.Get(assetId, &responseData); err != nil {

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

func (c Controller) FetchAssets(w http.ResponseWriter, r *http.Request) {

	var responseData []model.Asset
	apiResponse := utility.NewResponse()

	if err := c.Repository.Fetch(&responseData); err != nil {

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

func (c Controller) RemoveAsset(w http.ResponseWriter, r *http.Request) {

	apiResponse := utility.NewResponse()
	assetToRemove := model.Asset{}
	routeParams := mux.Vars(r)
	assetId, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.INPUTERROR, "Cannot cast Id"))
		return
	}
	assetToRemove.ID = assetId

	if err := c.Repository.Delete(&assetToRemove); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEMERROR {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, utility.GetCodeMsg(utility.SYSTEMERROR)))
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse.PlainError(utility.SYSTEMERROR, err.(utility.AppError).Error()))
		return
	}
	c.Logger.Info("Outgoing response to successful request %s", utility.GetCodeMsg(utility.SUCCESS))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.PlainSuccess(utility.SUCCESS, utility.GetCodeMsg(utility.SUCCESS)))

}
