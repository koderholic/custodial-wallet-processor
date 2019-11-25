package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// TxnType ...
type TxnType struct{ OFFCHAIN, ONCHAIN string }

// ProcessType ...
type ProcessType struct{ SINGLE, BATCH string }

// TxnTag ...
type TxnTag struct{ BUY, SELL, TRANSFER, DEPOSIT, WITHDRAW string }

// TxnStatus ...
type TxnStatus struct{ PENDING, PROCESSING, COMPLETED, TERMINATED, REVERSED string }

var (
	TransactionType = TxnType{
		OFFCHAIN: "Offchain",
		ONCHAIN:  "Onchain",
	}
	TransactionStatus = TxnStatus{
		PENDING:    "Pending",
		PROCESSING: "Processing",
		COMPLETED:  "Completed",
		TERMINATED: "Terminated",
		REVERSED:   "Reversed",
	}
	TransactionTag = TxnTag{
		BUY:      "Buy",
		SELL:     "Sell",
		TRANSFER: "Transfer",
		DEPOSIT:  "Deposit",
		WITHDRAW: "Withdraw",
	}

	ProcessingType = ProcessType{
		SINGLE: "Single",
		BATCH:  "Batch",
	}
)

//Transaction ... This is the transaction model for all user request
type Transaction struct {
	BaseModel
	AssetID              uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"assetId"`
	InitiatorID          uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"initiatorId"`
	Recipient            string    `json:"recipient"`
	TransactionReference string    `gorm:"not null;" json:"transactionReference"`
	TransactionType      string    `gorm:"not null;default:'Offchain'" json:"transactionType"`
	TransactionStatus    string    `gorm:"not null;default:'Pending'" json:"transactionStatus"`
	TransactionTag       string    `gorm:"not null;default:'Sell'" json:"transactionTag"`
	Volume               string    `gorm:"not null;default:'Sell'" json:"volume"`
	ReversedBalance      int64     `gorm:"type:BIGINT;not null" json:"reversedBalance"`
	ProcessingType       string    `gorm:"not null;default:'Single'" json:"processingType"`
	BatchID              uuid.UUID `gorm:"type:VARCHAR(36);" json:"batchId"`
	TransactionStartDate time.Time `json:"transactionStartDate"`
	TransactionEndDate   time.Time `json:"transactionEndDate"`
	BatchRequest         `sql:"-"`
}
