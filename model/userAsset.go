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

type Asset struct {
	AssetSymbol string  `json:"assetSymbol" validate:"required"`
	Volume      float64 `json:"volume" validate:"required"`
}

// CreditUserAssetRequest ... Model definition for credit user asset request
type CreditUserAssetRequest struct {
	Asset  Asset     `json:"asset" validate:"required"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}

// CreditUserAssetResponse ... Model definition for credit user asset request
type CreditUserAssetResponse struct {
	dto.Transaction
}
