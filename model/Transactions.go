package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type TxnType struct {
	OFFCHAIN, ONCHAIN string
}
type TxnStatus struct {
	PENDING, PROCESSING, COMPLETED, TERMINATED, REVERSED string
}
type TxnTag struct {
	BUY, SELL, TRANSFER, DEPOSIT, WITHDRAW string
}

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
)

type Transaction struct {
	BaseModel
	AssetId              uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"assetId"`
	InitiatorId          uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"initiatorId"`
	ChainTransactionId   uuid.UUID `gorm:"type:VARCHAR(36);" json:"chainTransactionId"`
	BatchId              uuid.UUID `gorm:"type:VARCHAR(36);" json:"batchId"`
	TransactionReference string    `gorm:"not null;" json:"transactionReference"`
	Recipient            string    `json:"recipient"`
	TransactionType      string    `gorm:"not null,default:'Offchain'" json:"transactionType"`
	TransactionStatus    string    `gorm:"not null,default:'Pending'" json:"transactionStatus"`
	TransactionTag       string    `gorm:"not null,default:'Sell'" json:"transactionTag"`
	Volume               string    `gorm:"not null,default:'Sell'" json:"transactionTag"`
	BookBalance          string    `gorm:"not null,default:'Sell'" json:"balance"`
	TransactionStartDate time.Time `json:"transactionStartDate"`
	transactionEndDate   time.Time `json:"transactionStartDate"`
	TokenType            string    `json:"tokenType"`
	Decimal              int       `json:"decimal"`
	IsEnabled            bool      `gorm:"default:1" json:"isEnabled"`
}
