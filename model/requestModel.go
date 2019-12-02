package model

import (
	"time"

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

// UpdateAuthTokenRequest ... Model definition for getting a new service auth token request
type UpdateAuthTokenRequest struct {
	ServiceID   string `json:"serviceId`
	Description string `json:"description`
}

// UpdateAuthTokenResponse ...
type UpdateAuthTokenResponse struct {
	ServiceID   string    `json:"serviceId`
	Description string    `json:"description`
	Permissions []string  `json:"permissions`
	CreatedAt   time.Time `json:"createdAt`
	ExpiresAt   time.Time `json:"expiresAt`
}
