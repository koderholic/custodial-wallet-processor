package model

import uuid "github.com/satori/go.uuid"

// ChainTransaction ... Model definition
type ChainTransaction struct {
	BaseModel
	Status          bool      `gorm:"index;not null;default:false" json:"status"`
	BatchID         uuid.UUID `gorm:"type:VARCHAR(36);index:batch_id" json:"batchId"`
	TransactionHash string    `json:"hash"`
	BlockHeight     int64     `gorm:"type:BIGINT" json:"blockHeight"`
	BatchRequest    `sql:"-"`
}
