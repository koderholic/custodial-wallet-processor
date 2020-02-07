package controllers

import (
	"encoding/json"
	"fmt"
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

// ExternalTransfer ...
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
	debitReferenceTransaction := dto.Transaction{}
	if err := controller.Repository.FetchByFieldName(&dto.Transaction{TransactionReference: requestData.DebitReference}, &debitReferenceTransaction); err != nil {
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

	// Checks to ensure the transaction status of debitReference is completed
	if debitReferenceTransaction.TransactionStatus != dto.TransactionStatus.COMPLETED {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", utility.INVALID_DEBIT)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INVALID_DEBIT", utility.INVALID_DEBIT))
		return
	}

	// Checks also that the value matches the value that was initially debited
	value := decimal.NewFromFloat(requestData.Value)
	debitValue, err := decimal.NewFromString(debitReferenceTransaction.Value)
	if err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}
	if value.GreaterThan(debitValue) {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INVALID_DEBIT_AMOUNT", utility.INVALID_DEBIT_AMOUNT))
		return
	}

	// Get asset associated with the debit reference
	debitReferenceAsset := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: debitReferenceTransaction.RecipientID}}, &debitReferenceAsset); err != nil {
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

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("Failed to complete external transafer on %s : %s", requestData.DebitReference, err)))
		return
	}
	// Create a transaction record for the transaction on the db for the request
	transaction := dto.Transaction{
		InitiatorID:          decodedToken.ServiceID,
		RecipientID:          debitReferenceTransaction.RecipientID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		DebitReference:       requestData.DebitReference,
		Memo:                 debitReferenceTransaction.Memo,
		TransactionType:      dto.TransactionType.ONCHAIN,
		TransactionTag:       dto.TransactionTag.WITHDRAW,
		Value:                value.String(),
		PreviousBalance:      debitReferenceTransaction.PreviousBalance,
		AvailableBalance:     debitReferenceTransaction.AvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Queue transaction up for processing
	queue := dto.TransactionQueue{
		Recipient:      requestData.RecipientAddress,
		Value:          transactionValue,
		DebitReference: requestData.DebitReference,
		Denomination:   debitReferenceAsset.Symbol,
		TransactionId:  transaction.ID,
	}
	if err := tx.Create(&queue).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	// Send acknowledgement to the calling service
	responseData.TransactionReference = transaction.TransactionReference
	responseData.DebitReference = requestData.DebitReference
	responseData.TransactionStatus = transaction.TransactionStatus

	controller.Logger.Info("Outgoing response to ExternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// ConfirmTransaction ...
func (controller UserAssetController) ConfirmTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.ChainData{}
	serviceErr := model.ServicesRequestErr{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for ConfirmTransaction : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	// Get the asset denoimiuntion associated with the transaction
	chainTransaction := dto.ChainTransaction{}
	err := controller.Repository.Get(&dto.ChainTransaction{TransactionHash: requestData.TransactionHash}, &chainTransaction)
	transactionDetails := dto.Transaction{}
	err = controller.Repository.Get(&dto.Transaction{OnChainTxId: chainTransaction.ID}, &transactionDetails)
	transactionQueueDetails := dto.TransactionQueue{}
	err = controller.Repository.GetByFieldName(&dto.TransactionQueue{TransactionId: transactionDetails.ID}, &transactionQueueDetails)
	if err != nil {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}

	// Calls TransactionStatus on crypto adapter to verify the transaction status
	// transactionStatusRequest := model.TransactionStatusRequest{
	// 	TransactionHash: requestData.TransactionHash,
	// 	AssetSymbol:     transactionQueueDetails.Denomination,
	// }
	transactionStatusResponse := model.TransactionStatusResponse{}
	// if err := services.TransactionStatus(controller.Logger, controller.Config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
	// 	controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v : %+v", serviceErr, err)
	// 	responseWriter.Header().Set("Content-Type", "application/json")
	// 	responseWriter.WriteHeader(http.StatusInternalServerError)
	// 	if serviceErr.Code != "" {
	// 		_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SVCS_KEYMGT_ERR", serviceErr.Message))
	// 		return
	// 	}
	// 	_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, err.Error())))
	// 	return
	// }

	if transactionStatusResponse.Status == "SUCCESS" {

		// Goes to chain transaction table, update the status of the chain transaction,
		chainTransactionUpdate := dto.ChainTransaction{Status: *requestData.Status, TransactionFee: requestData.TransactionFee, BlockHeight: requestData.BlockHeight}
		if err := tx.Model(&dto.ChainTransaction{TransactionHash: requestData.TransactionHash}).Updates(&chainTransactionUpdate).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
			return
		}

		// With the chainTransactionUpdateId it goes to the transactions table, fetches the transaction mapped to the chainId and updates the status
		transactionUpdate := dto.Transaction{TransactionStatus: dto.TransactionStatus.COMPLETED}
		if err := tx.Model(&dto.Transaction{OnChainTxId: chainTransactionUpdate.ID}).Updates(&transactionUpdate).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}
		// It goes to the queue table and fetches the queue matching the transactionId and updates the status to either TERMINATED or COMPLETED
		transactionQueueUpdate := dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.COMPLETED}
		if err := tx.Model(&dto.TransactionQueue{TransactionId: transactionUpdate.ID}).Updates(&transactionQueueUpdate).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}

	} else if transactionStatusResponse.Status == "FAILED" {

		// Goes to chain transaction table, update the status of the chain transaction,
		chainTransactionUpdate := dto.ChainTransaction{Status: *requestData.Status, TransactionFee: requestData.TransactionFee, BlockHeight: requestData.BlockHeight}
		if err := tx.Model(&dto.ChainTransaction{TransactionHash: requestData.TransactionHash}).Updates(&chainTransactionUpdate).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
			return
		}

		// With the chainTransactionUpdateId it goes to the transactions table, fetches the transaction mapped to the chainId and updates the status
		transactionUpdate := dto.Transaction{TransactionStatus: dto.TransactionStatus.TERMINATED}
		if err := tx.Model(&dto.Transaction{OnChainTxId: chainTransactionUpdate.ID}).Updates(&transactionUpdate).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}
		// It goes to the queue table and fetches the queue matching the transactionId and updates the status to either TERMINATED or COMPLETED
		transactionQueueUpdate := dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.TERMINATED}
		if err := tx.Model(&dto.TransactionQueue{TransactionId: transactionUpdate.ID}).Updates(&transactionQueueUpdate).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}

	} else {
		controller.Logger.Info("Outgoing response to ConfirmTransaction request %+v", apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	controller.Logger.Info("Outgoing response to ConfirmTransaction request %+v", apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))

}
