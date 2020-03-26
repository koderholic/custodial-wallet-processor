package model

import (
	uuid "github.com/satori/go.uuid"
)

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

// SignTransaction ... Request definition for sign transaction , key-management service
type SignTransactionRequest struct {
	FromAddress string `json:"fromAddress"`
	ToAddress   string `json:"toAddress"`
	Memo        string `json:"memo"`
	Amount      int64  `json:"amount"`
	AssetSymbol string `json:"assetSymbol"`
	IsSweep     bool   `json:"isSweep"`
}

// SignTransactionResponse ... Model definition for sign transaction successful response, key-management service
type SignTransactionResponse struct {
	SignedData string `json:"signedTransaction"`
	Fee        int64  `json:"fee"`
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

// TransactionStatusRequest ... Request definition for broadcast to chain , crypto-adapter service
type TransactionStatusRequest struct {
	TransactionHash string `json:"transactionHash"`
	AssetSymbol     string `json:"assetSymbol"`
}

// TransactionStatusResponse ... Model definition for broadcast to chain successful response, crypto-adapter service
type TransactionStatusResponse struct {
	TransactionHash string `json:"transactionHash"`
	Status          string `json:"status"`
	AssetSymbol     string `json:"assetSymbol"`
}

// LockerServiceRequest ... Request definition for  acquire or renew lock, locker service
type LockerServiceRequest struct {
	Identifier   string `json:"identifier"`
	Token        string `json:"token"`
	ExpiresAfter int64  `json:"expiresAfter"`
	Timeout      int64  `json:"timeout"`
}

// LockerServiceResponse ... Model definition for acquire lock successful response, locker service
type LockerServiceResponse struct {
	Identifier string `json:"identifier"`
	Token      string `json:"token"`
	ExpiresAt  string `json:"expiresAt"`
	Fence      int64  `json:"fence"`
}

// LockReleaseRequest ...Request definition for release lock, locker service
type LockReleaseRequest struct {
	Identifier string `json:"identifier"`
	Token      string `json:"token"`
}

// OnchainBalanceRequest ... Request definition for get on-chain balance, crypto-adapter service
type OnchainBalanceRequest struct {
	AssetSymbol string `json:"assetSymbol"`
	Address     string `json:"address"`
}

// OnchainBalanceResponse ... Model definition for get on-chain balance successful response, crypto-adapter service
type OnchainBalanceResponse struct {
	Balance     string `json:"balance"`
	AssetSymbol string `json:"assetSymbol"`
	Decimals    int    `json:"decimals"`
}
