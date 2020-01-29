package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// GetTransaction ... Retrieves the transaction details of the reference sent
func (controller BaseController) GetTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData model.TransactionResponse
	var transaction dto.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	transactionRef := routeParams["reference"]
	controller.Logger.Info("Incoming request details for GetTransaction : transaction reference : %+v", transactionRef)

	if err := controller.Repository.GetByFieldName(&dto.Transaction{TransactionReference: transactionRef}, &transaction); err != nil {
		controller.Logger.Error("Outgoing response to GetTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	transaction.Map(&responseData)

	controller.Logger.Info("Outgoing response to GetTransaction request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetTransactionsByAssetId ... Retrieves all transactions relating to an asset
func (controller BaseController) GetTransactionsByAssetId(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData model.TransactionListResponse
	var initiatorTransactions []dto.Transaction
	var recipientTransactions []dto.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}
	controller.Logger.Info("Incoming request details for GetTransactionsByAssetId : assetID : %+v", assetID)

	if err := controller.Repository.FetchByFieldName(&dto.Transaction{RecipientID: assetID}, &initiatorTransactions); err != nil {
		if err.Error() != utility.SQL_404 {
			controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}
	}

	if err := controller.Repository.GetByFieldName(&dto.Transaction{InitiatorID: assetID}, &recipientTransactions); err != nil {
		if err.Error() != utility.SQL_404 {
			controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}
	}

	for i := 0; i < len(initiatorTransactions); i++ {
		transaction := initiatorTransactions[i]
		tx := model.TransactionResponse{}

		transaction.Map(&tx)

		responseData.Transactions = append(responseData.Transactions, tx)

	}
	for i := 0; i < len(recipientTransactions); i++ {
		transaction := recipientTransactions[i]
		tx := model.TransactionResponse{}

		transaction.Map(&tx)

		responseData.Transactions = append(responseData.Transactions, tx)

	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []model.TransactionResponse{}
	}

	controller.Logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}
