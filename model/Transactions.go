package model

import (
	"time"
	"wallet-adapter/dto"

	uuid "github.com/satori/go.uuid"
)

// TxnType ...
type TxnType struct{ OFFCHAIN, ONCHAIN string }

// ProcessType ...
type ProcessType struct{ SINGLE, BATCH string }

// TxnTag ...
type TxnTag struct{ CREDIT, DEBIT, TRANSFER, DEPOSIT, WITHDRAW string }

// TxnStatus ...
type TxnStatus struct{ PENDING, PROCESSING, COMPLETED, TERMINATED, REJECTED string }

var (
	TransactionType = TxnType{
		OFFCHAIN: "OFFCHAIN",
		ONCHAIN:  "ONCHAIN",
	}
	TransactionStatus = TxnStatus{
		PENDING:    "PENDING",
		PROCESSING: "ONGOING",
		COMPLETED:  "COMPLETED",
		TERMINATED: "TERMINATED",
		REJECTED:   "REJECTED",
	}

	TransactionTag = TxnTag{
		CREDIT:   "CREDIT",
		DEBIT:    "DEBIT",
		TRANSFER: "TRANSFER",
		DEPOSIT:  "DEPOSIT",
		WITHDRAW: "WITHDRAW",
	}

	ProcessingType = ProcessType{
		SINGLE: "SINGLE",
		BATCH:  "BATCH",
	}
)

//Transaction ... This is the transaction DTO for all user request
type Transaction struct {
	BaseModel
	InitiatorID          uuid.UUID    `gorm:"type:VARCHAR(36);not null;index:initiator_id" json:"initiator_id,omitempty"`
	RecipientID          uuid.UUID    `json:"type:VARCHAR(36);not null" json:"recipient_id,omitempty"`
	TransactionReference string       `gorm:"not null;unique_index" json:"transaction_reference,omitempty"`
	PaymentReference     string       `gorm:"not null;unique_index" json:"payment_reference,omitempty"`
	DebitReference       string       `json:"debit_reference,omitempty"`
	Memo                 string       `gorm:"not null;" json:"memo,omitempty"`
	TransactionType      string       `gorm:"not null;default:'OFFCHAIN'" json:"transaction_type,omitempty"`
	TransactionStatus    string       `gorm:"not null;default:'PENDING';index:transaction_status" json:"transaction_status,omitempty"`
	TransactionTag       string       `gorm:"not null;default:'CREDIT'" json:"transaction_tag,omitempty"`
	Value                string       `gorm:"type:decimal(64,18);not null" json:"value,omitempty"`
	PreviousBalance      string       `gorm:"type:decimal(64,18);not null" json:"previous_balance,omitempty"`
	AvailableBalance     string       `gorm:"type:decimal(64,18);not null" json:"available_balance,omitempty"`
	ProcessingType       string       `gorm:"not null;default:'SINGLE'" json:"processing_type,omitempty"`
	OnChainTxId          uuid.UUID    `gorm:"type:VARCHAR(36);" json:"on_chain_tx_id,omitempty"`
	BatchID              uuid.UUID    `gorm:"type:VARCHAR(36);" json:"batch_id,omitempty"`
	TransactionStartDate time.Time    `json:"transaction_start_date,omitempty"`
	TransactionEndDate   time.Time    `json:"transaction_end_date,omitempty"`
	SweptStatus          bool         `gorm:"not null;default:false" json:"swept_status,omitempty"`
	Batch                BatchRequest `sql:"-" json:"omitempty"`
	AssetSymbol          string       `gorm:"type:VARCHAR(36);not null;" json:"asset_symbol,omitempty"`
	Network          string       `json:"network,omitempty"`
}

func (transaction Transaction) Map(tx *dto.TransactionResponse) {
	tx.ID = transaction.ID
	tx.InitiatorID = transaction.InitiatorID
	tx.RecipientID = transaction.RecipientID
	tx.Value = transaction.Value
	tx.TransactionStatus = transaction.TransactionStatus
	tx.TransactionReference = transaction.TransactionReference
	tx.PaymentReference = transaction.PaymentReference
	tx.PreviousBalance = transaction.PreviousBalance
	tx.AvailableBalance = transaction.AvailableBalance
	tx.TransactionType = transaction.TransactionType
	tx.TransactionEndDate = transaction.TransactionEndDate
	tx.TransactionStartDate = transaction.TransactionStartDate
	tx.CreatedDate = transaction.CreatedAt
	tx.UpdatedDate = transaction.UpdatedAt
	tx.TransactionTag = transaction.TransactionTag
}
