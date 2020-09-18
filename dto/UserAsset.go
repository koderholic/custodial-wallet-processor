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
