package model

import (
	uuid "github.com/satori/go.uuid"
)

//TransactionQueue ... This is the transaction DTO for all queued transactions for processing
type TransactionQueue struct {
	BaseModel
	Sender            string    `json:"sender,omitempty"`
	Recipient         string    `gorm:"not null" json:"recipient,omitempty"`
	Value             int64     `gorm:"type:bigint;not null" json:"value,omitempty"`
	Memo              string    `gorm:"type:VARCHAR(300);" json:"memo,omitempty"`
	AssetSymbol       string    `gorm:"type:VARCHAR(36);not null;" json:"asset_symbol,omitempty"`
	DebitReference    string    `gorm:"type:VARCHAR(150);not null;unique_index" json:"debit_reference,omitempty"`
	TransactionId     uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"transaction_id,omitempty"`
	BatchID     uuid.UUID `gorm:"type:VARCHAR(36);" json:"batch_id,omitempty"`
	TransactionStatus string    `gorm:"not null;default:'PENDING'" json:"transaction_status,omitempty"`
}
