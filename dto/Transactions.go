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
type TxnTag struct{ CREDIT, DEBIT, TRANSFER, DEPOSIT, WITHDRAW string }

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
		CREDIT:   "Credit",
		DEBIT:    "Debit",
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
	InitiatorID          uuid.UUID    `gorm:"type:VARCHAR(36);not null;index:initiator_id" json:"initiator_id,omitempty"`
	RecipientID          uuid.UUID    `json:"recipient,omitempty"`
	TransactionReference string       `gorm:"not null;unique_index" json:"transaction_reference,omitempty"`
	PaymentReference     string       `gorm:"not null;unique_index" json:"payment_reference,omitempty"`
	Memo                 string       `gorm:"not null;" json:"memo,omitempty"`
	TransactionType      string       `gorm:"not null;default:'Offchain'" json:"transaction_type,omitempty"`
	TransactionStatus    string       `gorm:"not null;default:'Pending';index:transaction_status" json:"transaction_status,omitempty"`
	TransactionTag       string       `gorm:"not null;default:'Credit'" json:"transaction_tag,omitempty"`
	Value                string       `gorm:"type:decimal(64,18);not null" json:"value,omitempty"`
	PreviousBalance      string       `gorm:"type:decimal(64,18);not null" json:"previous_balance,omitempty"`
	AvailableBalance     string       `gorm:"type:decimal(64,18);not null" json:"available_balance,omitempty"`
	ProcessingType       string       `gorm:"not null;default:'Single'" json:"processing_type,omitempty"`
	BatchID              uuid.UUID    `gorm:"type:VARCHAR(36);" json:"batch_id,omitempty"`
	TransactionStartDate time.Time    `json:"transaction_start_date,omitempty"`
	TransactionEndDate   time.Time    `json:"transaction_end_date,omitempty"`
	Batch                BatchRequest `sql:"-" json:"omitempty"`
}
