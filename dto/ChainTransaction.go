package dto

import uuid "github.com/satori/go.uuid"

// ChainTransaction ... DTO definition for on-chain transactions
type ChainTransaction struct {
	BaseDTO
	Status          string    `gorm:"index;not null;default:PENDING" json:"status"`
	BatchID         uuid.UUID `gorm:"type:VARCHAR(36);index:batch_id" json:"batch_id"`
	TransactionHash string    `json:"hash"`
	TransactionFee  string    `json:"TransactionFee"`
	BlockHeight     int64     `gorm:"type:BIGINT" json:"block_height"`
	BatchRequest    `sql:"-"`
}
