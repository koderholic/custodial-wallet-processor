package dto

import (
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

// CreditUserAssetRequest ... Model definition for credit user asset request
type InternalTransferRequest struct {
	InitiatorAssetId     uuid.UUID `json:"initiatorAssetId" validate:"required"`
	RecipientAssetId     uuid.UUID `json:"recipientAssetId" validate:"required"`
	Value                float64   `json:"value" validate:"required"`
	TransactionReference string    `json:"transactionReference" validate:"required"`
	Memo                 string    `json:"memo" validate:"required"`
}

// CreditUserAssetRequest ... Model definition for credit user asset request
type CreditUserAssetRequest struct {
	AssetID              uuid.UUID `json:"assetId" validate:"required"`
	Value                float64   `json:"value" validate:"required"`
	TransactionReference string    `json:"transactionReference" validate:"required"`
	Memo                 string    `json:"memo"`
}

// OnChainCreditUserAssetRequest object
type OnChainCreditUserAssetRequest struct {
	CreditUserAssetRequest
	ChainData ChainData `json:"chainData" validate:"required"`
}

// ChainData On-chain metadata for broadcasted / incoming transactions
type ChainData struct {
	Status           *bool  `json:"status" validate:"required"`
	TransactionHash  string `json:"transactionHash" validate:"required"`
	TransactionFee   string `json:"transactionFee" validate:"required"`
	RecipientAddress string `json:"recipientAddress"`
	BlockHeight      int64  `json:"blockHeight"`
}
