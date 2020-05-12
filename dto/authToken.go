package dto

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
	ServiceID   string   `json:"serviceId`
	Token       string   `json:"token`
	Permissions []string `json:"permissions`
	CreatedAt   string   `json:"createdAt`
	ExpiresAt   string   `json:"expiresAt`
}
