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

	responseData.ID = transaction.ID
	responseData.InitiatorID = transaction.InitiatorID
	responseData.RecipientID = transaction.RecipientID
	responseData.Value = transaction.Value
	responseData.TransactionStatus = transaction.TransactionStatus
	responseData.TransactionReference = transaction.TransactionReference
	responseData.PaymentReference = transaction.PaymentReference
	responseData.PreviousBalance = transaction.PreviousBalance
	responseData.AvailableBalance = transaction.AvailableBalance
	responseData.TransactionType = transaction.TransactionType
	responseData.TransactionEndDate = transaction.TransactionEndDate
	responseData.TransactionStartDate = transaction.TransactionStartDate
	responseData.CreatedDate = transaction.CreatedAt
	responseData.UpdatedDate = transaction.UpdatedAt
	responseData.TransactionTag = transaction.TransactionTag

	controller.Logger.Info("Outgoing response to GetTransaction request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetTransactionsByAssetId ... Retrieves all transactions relating to an asset
func (controller BaseController) GetTransactionsByAssetId(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData model.TransactionListResponse
	var transactions []dto.Transaction
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

	if err := controller.Repository.FetchByFieldName(&dto.Transaction{RecipientID: assetID}, &transactions); err != nil {

		if err.Error() == utility.SQL_404 {
			errController := controller.Repository.GetByFieldName(&dto.Transaction{InitiatorID: assetID}, &transactions)
			if errController == nil {
				return
			}
		}
		controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		tx := model.TransactionResponse{}

		tx.ID = transaction.ID
		tx.InitiatorID = transaction.InitiatorID
		tx.RecipientID = transaction.RecipientID
		tx.Value = transaction.Value
		tx.TransactionStatus = transaction.TransactionStatus
		tx.TransactionReference = transaction.TransactionReference
		tx.PaymentReference = transaction.PaymentReference
		tx.PreviousBalance = transaction.PreviousBalance
		tx.AvailableBalance = transaction.AvailableBalance
		tx.TransactionType = transaction.TransactionType
		tx.TransactionEndDate = transaction.TransactionEndDate
		tx.TransactionStartDate = transaction.TransactionStartDate
		tx.CreatedDate = transaction.CreatedAt
		tx.UpdatedDate = transaction.UpdatedAt
		tx.TransactionTag = transaction.TransactionTag

		responseData.Transactions = append(responseData.Transactions, tx)

	}

	controller.Logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}
