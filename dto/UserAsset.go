package dto

import (
	uuid "github.com/satori/go.uuid"
)

// CreateUserAssetRequest ... Model definition for create asset request
type CreateUserAssetRequest struct {
	Assets []string  `json:"assets" validate:"required,gt=0"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}

type Asset struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"userId"`
	AssetSymbol      string    `json:"symbol"`
	AvailableBalance string    `json:"availableBalance"`
	Decimal          int       `json:"decimal"`
}

// CreateUserAssetResponse ... Model definition for create asset response
type UserAssetResponse struct {
	Assets []Asset `json:"assets"`
}

// CreditUserAssetRequest ... Model definition for credit user asset request
type CreditUserAssetRequest struct {
	AssetID              uuid.UUID `json:"assetId" validate:"required"`
	Value                float64   `json:"value" validate:"required"`
	TransactionReference string    `json:"transactionReference" validate:"required"`
	Memo                 string    `json:"memo"`
}

type ChainData struct {
	Status           *bool  `json:"status" validate:"required"`
	TransactionHash  string `json:"transactionHash" validate:"required"`
	TransactionFee   string `json:"transactionFee" validate:"required"`
	RecipientAddress string `json:"recipientAddress"`
	BlockHeight      int64  `json:"blockHeight"`
}

type OnChainCreditUserAssetRequest struct {
	CreditUserAssetRequest
	ChainData ChainData `json:"chainData" validate:"required"`
}

// CreditUserAssetRequest ... Model definition for credit user asset request
type InternalTransferRequest struct {
	InitiatorAssetId     uuid.UUID `json:"initiatorAssetId" validate:"required"`
	RecipientAssetId     uuid.UUID `json:"recipientAssetId" validate:"required"`
	Value                float64   `json:"value" validate:"required"`
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

type AddressWithMemo struct {
	Address string `json:"address,omitempty"`
	Memo    string `json:"memo,omitempty"`
}
