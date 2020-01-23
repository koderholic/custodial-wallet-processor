package model

import (
	uuid "github.com/satori/go.uuid"
)

// CreateUserAssetRequest ... Model definition for create asset request
type GenerateAddressRequest struct {
	UserID uuid.UUID `json:"userId"`
	Symbol string    `json:"symbol"`
}

// GenerateAddressResponse ... Model definition for key management external services request made with successful response
type GenerateAddressResponse struct {
	Address string    `json:"address"`
	UserID  uuid.UUID `json:"userId"`
}

// ServicesRequestErr ... Model definition for external services request made with error response
type ServicesRequestErr struct {
	Success bool              `json:"success"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

// ServicesRequestSuccess ... Model definition for external services request made with successful response but no data
type ServicesRequestSuccess struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}
