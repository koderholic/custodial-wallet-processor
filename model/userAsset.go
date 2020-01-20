package model

import (
	"wallet-adapter/dto"

	uuid "github.com/satori/go.uuid"
)

// CreateUserAssetRequest ... Model definition for create asset request
type CreateUserAssetRequest struct {
	Assets []string  `json:"assets" validate:"required,gt=0"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}

// CreateUserAssetResponse ... Model definition for create asset response
type CreateUserAssetResponse struct {
	Assets []dto.UserBalance `json:"assets"`
}

// CreditUserAssetRequest ... Model definition for credit user asset request
type CreditUserAssetRequest struct {
	AssetID              uuid.UUID `json:"assetId" validate:"required"`
	Value                string    `json:"value" validate:"required"`
	TransactionReference string    `json:"transactionReference" validate:"required"`
	Memo                 string    `json:"memo" validate:"required"`
}

// CreditUserAssetRequest ... Model definition for credit user asset request
type InternalTransferRequest struct {
	InitiatorAssetId     uuid.UUID `json:"initiatorAssetId" validate:"required"`
	RecipientAssetId     uuid.UUID `json:"recipientAssetId" validate:"required"`
	Value                string    `json:"value" validate:"required"`
	TransactionReference string    `json:"transactionReference" validate:"required"`
	Memo                 string    `json:"memo" validate:"required"`
}

// TransactionReceipt ... Model definition for credit user asset request
type TransactionReceipt struct {
	AssetID              uuid.UUID `json:"assetId,omitempty"`
	Value                string    `json:"value,omitempty"`
	TransactionReference string    `json:"transactionReference,omitempty"`
	PaymentReference     string    `json:"paymentReference,omitempty"`
	TransactionStatus    string    `json:"transactionStatus,omitempty"`
}
