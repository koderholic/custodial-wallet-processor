package model

import (
	uuid "github.com/satori/go.uuid"
)

// GenerateAddressRequest ... Request definition for generate address , key-management service
type GenerateAddressRequest struct {
	UserID uuid.UUID `json:"userId"`
	Symbol string    `json:"symbol"`
}

// GenerateAddressResponse ... Model definition for generate address successful response, key-management service
type GenerateAddressResponse struct {
	Address string    `json:"address"`
	UserID  uuid.UUID `json:"userId"`
}

// SignTransaction ... Request definition for sign transaction , key-management service
type SignTransactionRequest struct {
	FromAddress string `json:"fromAddress"`
	ToAddress   string `json:"toAddress"`
	Amount      int64  `json:"amount"`
	CoinType    string `json:"coinType"`
}

// SignTransactionResponse ... Model definition for sign transaction successful response, key-management service
type SignTransactionResponse struct {
	SignedData string `json:"signedData"`
}

// BroadcastToChainRequest ... Request definition for broadcast to chain , crypto-adapter service
type BroadcastToChainRequest struct {
	SignedData  string `json:"signedData"`
	AssetSymbol string `json:"assetSymbol"`
}

// BroadcastToChainResponse ... Model definition for broadcast to chain successful response, crypto-adapter service
type BroadcastToChainResponse struct {
	TransactionHash string `json:"transactionHash"`
	Error           bool   `json:"error"`
	Message         string `json:"message"`
}

type SubscriptionRequest struct {
	Subscriptions map[string][]string `json:"subscriptions"`
	Webhook       string              `json:"webhook"`
}

type SubscriptionResponse struct {
	Message string `json:"message"`
	Status  bool   `json:"status"`
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
