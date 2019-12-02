package model

// TokenType ...
type TokenType struct {
	SERVICE, USER string
}

//JWT_TOKEN_TYPE ...
var JWT_TOKEN_TYPE = TokenType{
	SERVICE: "SERVICE",
	USER:    "USER",
}

// TokenClaims ... Model definition for jwt token  claims
type TokenClaims struct {
	TokenType   string   `json:"tokenType,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}
