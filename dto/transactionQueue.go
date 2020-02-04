package dto

import (
	uuid "github.com/satori/go.uuid"
)

//TransactionQueue ... This is the transaction DTO for all queued transactions for processing
type TransactionQueue struct {
	BaseDTO
	Sender            string    `json:"sender,omitempty"`
	Recipient         string    `gorm:"not null" json:"recipient,omitempty"`
	Value             int64     `gorm:"type:decimal(64,18);not null" json:"value,omitempty"`
	Denomination      string    `gorm:"type:VARCHAR(36);not null;" json:"denomination,omitempty"`
	DebitReference    string    `gorm:"type:VARCHAR(150);not null;unique_index" json:"debit_reference,omitempty"`
	TransactionId     uuid.UUID `gorm:"type:VARCHAR(36);not null;index:transaction_id" json:"transaction_id,omitempty"`
	TransactionStatus string    `gorm:"not null;default:'PENDING';index:transaction_status" json:"transaction_status,omitempty"`
}
