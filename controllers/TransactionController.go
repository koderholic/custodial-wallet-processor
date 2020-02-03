package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
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

	if err := controller.Repository.FetchByFieldName(&dto.Transaction{InitiatorID: assetID}, &initiatorTransactions); err != nil {
		if err.Error() != utility.SQL_404 {
			controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}
	}

	if err := controller.Repository.FetchByFieldName(&dto.Transaction{RecipientID: assetID}, &recipientTransactions); err != nil {
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
		receipientTransaction := recipientTransactions[i]
		txRecipient := model.TransactionResponse{}

		receipientTransaction.Map(&txRecipient)

		responseData.Transactions = append(responseData.Transactions, txRecipient)

	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []model.TransactionResponse{}
	}

	controller.Logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// ExternalTransfer ... Bundle-406
func (controller UserAssetController) ExternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.ExternalTransferRequest{}
	responseData := model.ExternalTransferResponse{}
	paymentRef := utility.RandomString(16)

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for ExternalTransfer : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	// A check is done to ensure the debitReference points to an actual previous debit
	debitReference := dto.Transaction{}
	if err := controller.Repository.FetchByFieldName(&dto.Transaction{TransactionReference: requestData.DebitReference}, &debitReference); err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}
	//	Ensure the debit reference has not already been processed, as the debitRefence field will be populated if processed
	if debitReference.DebitReference != "" {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", utility.DEBIT_PROCESSED_ERR)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INVALID_DEBIT", utility.DEBIT_PROCESSED_ERR))
		return
	}

	// Checks to ensure the transaction status of debitReference is completed
	if debitReference.TransactionStatus != dto.TransactionStatus.COMPLETED {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", utility.INVALID_DEBIT)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INVALID_DEBIT", utility.INVALID_DEBIT))
		return
	}

	// Checks also that the value matches the value that was initially debited
	value := decimal.NewFromFloat(requestData.Value)
	debitValue, err := decimal.NewFromString(debitReference.Value)
	if err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}
	if value.LessThan(debitValue) {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INVALID_DEBIT_AMOUNT", utility.INVALID_DEBIT_AMOUNT))
		return
	}

	// Get asset associated with the debit reference
	debitReferenceAsset := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: debitReference.RecipientID}}, &debitReferenceAsset); err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Convert value to crypto smallest unit
	denominationDecimal := decimal.NewFromInt(int64(debitReferenceAsset.Decimal))
	baseExp := decimal.NewFromInt(10)
	transactionValue, err := strconv.ParseInt(value.Mul(baseExp.Pow(denominationDecimal)).String(), 10, 64)
	if err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	// Create a transaction record for the transaction on the db for the request
	transaction := dto.Transaction{
		InitiatorID:          decodedToken.ServiceID,
		RecipientID:          debitReference.RecipientID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 debitReference.Memo,
		TransactionType:      dto.TransactionType.ONCHAIN,
		TransactionTag:       dto.TransactionTag.DEBIT,
		Value:                value.String(),
		PreviousBalance:      debitReference.PreviousBalance,
		AvailableBalance:     debitReference.AvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
	}
	if err := controller.Repository.Create(&transaction); err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Queue transaction up for processing
	queue := dto.TransactionQueue{
		Recipient:     requestData.RecipientAddress,
		Value:         transactionValue,
		Denomination:  debitReferenceAsset.Symbol,
		TransactionId: transaction.ID,
	}
	if err := controller.Repository.Create(&queue); err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}

	// Acknowledge transaction to the calling service
	responseData.TransactionReference = transaction.TransactionReference
	responseData.DebitReference = requestData.DebitReference
	responseData.TransactionStatus = transaction.TransactionStatus

	controller.Logger.Info("Outgoing response to ExternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}
