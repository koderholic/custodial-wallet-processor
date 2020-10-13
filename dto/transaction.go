package dto

import (
	"math/big"
	"time"

	uuid "github.com/satori/go.uuid"
)

// TransactionResponse ... Model definition for get transaction request
type TransactionResponse struct {
	ID                   uuid.UUID  `json:"id,omitempty"`
	InitiatorID          uuid.UUID  `json:"initiatorId,omitempty"`
	RecipientID          uuid.UUID  `json:"recipientId,omitempty"`
	Value                string     `json:"value,omitempty"`
	TransactionStatus    string     `json:"transactionStatus,omitempty"`
	TransactionReference string     `json:"transactionReference,omitempty"`
	PaymentReference     string     `json:"paymentReference,omitempty"`
	PreviousBalance      string     `json:"previousBalance,omitempty"`
	AvailableBalance     string     `json:"availableBalance,omitempty"`
	TransactionType      string     `json:"transactionType,omitempty"`
	TransactionEndDate   time.Time  `json:"transactionEndDate,omitempty"`
	TransactionStartDate time.Time  `json:"transactionStartDate,omitempty"`
	CreatedDate          time.Time  `json:"createdDate,omitempty"`
	UpdatedDate          time.Time  `json:"updatedDate,omitempty"`
	TransactionTag       string     `json:"transactionTag,omitempty"`
	ChainData            *ChainData `json:"chainData"`
}

type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions,omitempty"`
}

// TransactionReceipt ... Model definition for credit user asset request
type TransactionReceipt struct {
	AssetID              uuid.UUID `json:"assetId,omitempty"`
	Value                string    `json:"value,omitempty"`
	TransactionReference string    `json:"transactionReference,omitempty"`
	PaymentReference     string    `json:"paymentReference,omitempty"`
	TransactionStatus    string    `json:"transactionStatus,omitempty"`
}

type ExternalTransferRequest struct {
	RecipientAddress     string  `json:"recipientAddress,omitempty" validate:"required"`
	Value                float64 `json:"value,omitempty" validate:"required"`
	DebitReference       string  `json:"debitReference,omitempty" validate:"required"`
	TransactionReference string  `json:"transactionReference,omitempty" validate:"required"`
}

type ExternalTransferResponse struct {
	TransactionReference string `json:"transactionReference,omitempty"`
	DebitReference       string `json:"debitReference,omitempty"`
	TransactionStatus    string `json:"transactionStatus,omitempty"`
}

// SignTransaction ... Request definition for sign transaction , key-management service
type SignTransactionRequest struct {
	FromAddress string   `json:"fromAddress"`
	ToAddress   string   `json:"toAddress"`
	Memo        string   `json:"memo"`
	Amount      *big.Int `json:"amount"`
	AssetSymbol string   `json:"assetSymbol"`
	IsSweep     bool     `json:"isSweep"`
	ProcessType string   `json:"processType"`
	Reference   string   `json:"reference"`
}

// SignTransactionResponse ... Model definition for sign transaction successful response, key-management service
type SignTransactionResponse struct {
	SignedData string `json:"signedTransaction"`
	Fee        int64  `json:"fee"`
}

// SignAndBroadcastResponse ... Model definition for broadcast to chain successful response, crypto-adapter service
type SignAndBroadcastResponse struct {
	TransactionHash string `json:"transactionHash"`
}

// TransactionStatusRequest ... Request definition for broadcast to chain , crypto-adapter service
type TransactionStatusRequest struct {
	TransactionHash string `json:"transactionHash"`
	AssetSymbol     string `json:"assetSymbol"`
	Reference       string `json:"reference"`
}

// TransactionStatusResponse ... Model definition for broadcast to chain successful response, crypto-adapter service
type TransactionStatusResponse struct {
	TransactionHash       string `json:"transactionHash"`
	Status                string `json:"status"`
	AssetSymbol           string `json:"assetSymbol"`
	TransactionFee        string `json:"fee"`
	BlockHeight           string `json:"height"`
	LastTimeStatusFetched string `json:"lastTimeStatusFetched"`
}

type TransactionListInfo struct {
	Decimal     int
	AssetSymbol string
}
