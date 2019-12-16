package dto

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

//Transaction ... This is the transaction DTO for all user request
type Transaction struct {
	BaseDTO
	AssetID              uuid.UUID    `gorm:"type:VARCHAR(36);not null" json:"asset_id"`
	InitiatorID          uuid.UUID    `gorm:"type:VARCHAR(36);not null;index:initiator_id" json:"initiator_id"`
	Recipient            string       `json:"recipient"`
	TransactionReference string       `gorm:"not null;" json:"transaction_reference"`
	TransactionType      string       `gorm:"not null;default:'Offchain'" json:"transaction_type"`
	TransactionStatus    string       `gorm:"not null;default:'Pending';index:transaction_status" json:"transaction_status"`
	TransactionTag       string       `gorm:"not null;default:'Sell'" json:"transaction_tag"`
	Volume               string       `gorm:"not null;default:'Sell'" json:"volume"`
	ReversedBalance      int64        `gorm:"type:BIGINT;not null" json:"reversed_balance"`
	ProcessingType       string       `gorm:"not null;default:'Single'" json:"processing_type"`
	BatchID              uuid.UUID    `gorm:"type:VARCHAR(36);" json:"batch_id"`
	TransactionStartDate time.Time    `json:"transaction_start_date"`
	TransactionEndDate   time.Time    `json:"transaction_end_date"`
	Batch                BatchRequest `sql:"-"`
}
