package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// transactionRequest ... Model definition for get transaction request
type TransactionResponse struct {
	ID                   uuid.UUID `json:"id,omitempty"`
	InitiatorID          uuid.UUID `json:"initiatorId,omitempty"`
	RecipientID          uuid.UUID `json:"recipientId,omitempty"`
	Value                string    `json:"value,omitempty"`
	TransactionStatus    string    `json:"transactionStatus,omitempty"`
	TransactionReference string    `json:"transactionReference,omitempty"`
	PaymentReference     string    `json:"paymentReference,omitempty"`
	PreviousBalance      string    `json:"previousBalance,omitempty"`
	AvailableBalance     string    `json:"availableBalance,omitempty"`
	TransactionType      string    `json:"transactionType,omitempty"`
	TransactionEndDate   time.Time `json:"transactionEndDate,omitempty"`
	TransactionStartDate time.Time `json:"transactionStartDate,omitempty"`
	CreatedDate          time.Time `json:"createdDate,omitempty"`
	UpdatedDate          time.Time `json:"updatedDate,omitempty"`
	TransactionTag       string    `json:"transactionTag,omitempty"`
}

type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions,omitempty"`
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

type BatchBTCRequest struct {
	AssetSymbol   string            `json:"assetSymbol"`
	ChangeAddress string            `json:"changeAddress"`
	IsSweep       bool              `json:"isSweep"`
	Origins       []string          `json:"origins"`
	Recipients    []BatchRecipients `json:"recipients"`
}

type BatchRecipients struct {
	Address string `json:"address"`
	Value   int    `json:"value"`
}
