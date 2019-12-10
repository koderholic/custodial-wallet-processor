package model

import (
	"time"

	"wallet-adapter/dto"

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
	Assets []dto.UserBalance `json:"assets"`
	Errors []string          `json:"errors"`
}

// AuthTokenRequestBody ...
type AuthTokenRequestBody struct {
	ServiceID string `json:"serviceId,omitempty`
	Payload   string `json:"payload,omitempty`
}

// UpdateAuthTokenRequest ... Model definition for getting a new service auth token request
type UpdateAuthTokenRequest struct {
	Body AuthTokenRequestBody `json:"body"`
}

// UpdateAuthTokenResponse ...
type UpdateAuthTokenResponse struct {
	ServiceID   string        `json:"serviceId`
	Token       string        `json:"token`
	Permissions []string      `json:"permissions`
	CreatedAt   time.Time     `json:"createdAt`
	ExpiresAt   time.Duration `json:"expiresAt`
}
