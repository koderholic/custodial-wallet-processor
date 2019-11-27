package model

import (
	uuid "github.com/satori/go.uuid"
	// "gopkg.in/go-playground/validator.v9"
)

// CreateUserAssetRequest ... Model definition for create asset request
type CreateUserAssetRequest struct {
	Assets []string  `json:"assets" validate:"required"`
	UserID uuid.UUID `json:"userId" validate:"required"`
}

// CreateUserAssetResponse ... Model definition for create asset response
type CreateUserAssetResponse struct {
	Assets []UserBalance `json:"assets"`
	Errors []string      `json:"errors"`
}
