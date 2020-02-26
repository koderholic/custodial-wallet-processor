package dto

import uuid "github.com/satori/go.uuid"

// ChainTransaction ... DTO definition for on-chain transactions
type ChainTransaction struct {
	BaseDTO
	Status          bool      `gorm:"index;not null;default:false" json:"status"`
	BatchID         uuid.UUID `gorm:"type:VARCHAR(36);index:batch_id" json:"batch_id"`
	TransactionHash string    `json:"transaction_hash"`
	TransactionFee  string    `json:"transaction_fee"`
	AssetSymbol     string    `json:"asset_symbol"`
	BlockHeight     int64     `gorm:"type:BIGINT" json:"block_height"`
	BatchRequest    `sql:"-"`
}
