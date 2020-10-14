package dto

import uuid "github.com/satori/go.uuid"

type AssetAddress struct {
	Address string `json:"address,omitempty"`
	Memo    string `json:"memo,omitempty"`
	Type    string `json:"type,omitempty"`
}

type AllAssetAddresses struct {
	Addresses          []AssetAddress `json:"addresses"`
	DefaultAddressType string         `json:"defaultAddressType"`
}

// GenerateAddressRequest ... Request definition for generate address , key-management service
type GenerateAddressRequest struct {
	UserID      uuid.UUID `json:"userId"`
	AssetSymbol string    `json:"symbol"`
}

// GenerateAddressResponse ... Model definition for generate address successful response, key-management service
type GenerateAddressResponse struct {
	Address string    `json:"address"`
	UserID  uuid.UUID `json:"userId"`
}

type AllAddressResponse struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// GenerateAllAddressesResponse ... Model definition for generate all asset addresses successful response, key-management service
type GenerateAllAddressesResponse struct {
	Addresses []AllAddressResponse `json:"addresses"`
	UserID    uuid.UUID            `json:"userId"`
}

type SubscriptionRequestV2 struct {
	Subscriptions map[string][]string `json:"subscriptions"`
}

type SubscriptionResponse struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
}
