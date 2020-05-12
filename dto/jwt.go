package dto

import uuid "github.com/satori/go.uuid"

// TokenType ...
type TokenType struct {
	SERVICE, USER string
}

//JWT_TOKEN_TYPE ...
var JWT_TOKEN_TYPE = TokenType{
	SERVICE: "SERVICE",
	USER:    "USER",
}

// ISSUER
var JWT_ISSUER = "SVCS/AUTH"

// TokenClaims ... Model definition for jwt token  claims
type TokenClaims struct {
	TokenType   string    `json:"tokenType,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	ServiceID   uuid.UUID `json:"serviceId,omitempty"`
	IAT         string    `json:"iat"`
	EXP         string    `json:"exp"`
	ISS         string    `json:"iss"`
}
