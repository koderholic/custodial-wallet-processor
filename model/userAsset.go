package model

import (
	"time"
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

type Asset struct {
	AssetSymbol string  `json:"assetSymbol" validate:"required"`
	Value       float64 `json:"value" validate:"required"`
}

// CreditUserAssetRequest ... Model definition for credit user asset request
type CreditUserAssetRequest struct {
	Asset  Asset     `json:"asset" validate:"required"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}

// CreditUserAssetResponse ... Model definition for credit user asset request
type CreditUserAssetResponse struct {
	ID                   uuid.UUID `json:"id,omitempty"`
	Asset                string    `json:"asset,omitempty"`
	InitiatorID          uuid.UUID `json:"initiatorId,omitempty"`
	RecipientID          uuid.UUID `json:"recipientId,omitempty"`
	TransactionReference string    `json:"transactionReference,omitempty"`
	TransactionType      string    `json:"transactionType,omitempty"`
	TransactionStatus    string    `json:"transactionStatus,omitempty"`
	TransactionTag       string    `json:"transactionTag,omitempty"`
	Value                float64   `json:"value,omitempty"`
	PreviousBalance      float64   `json:"previousBalance,omitempty"`
	AvailableBalance     float64   `json:"availableBalance,omitempty"`
	ReservedBalance      float64   `json:"reservedBalance,omitempty"`
	ProcessingType       string    `json:"processingType,omitempty"`
	TransactionStartDate time.Time `json:"transactionStart_date,omitempty"`
	TransactionEndDate   time.Time `json:"transactionEndDate,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}
