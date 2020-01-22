package model

import (
	uuid "github.com/satori/go.uuid"
)

// CreateUserAssetRequest ... Model definition for create asset request
type GenerateAddressRequest struct {
	UserID uuid.UUID `json:"userId"`
	Symbol string    `json:"symbol"`
}

// CreateUserAssetResponse ... Model definition for create asset response
type GenerateAddressResponse struct {
	Success bool              `json:"success"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}
