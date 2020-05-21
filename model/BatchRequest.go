package model

import (
	"time"
)

// BTHStatus ...
type BTHStatus struct {
	WAIT_MODE, RETRY_MODE, PROCESSING, COMPLETED, TERMINATED string
}

var (
	// BatchStatus ...
	BatchStatus = BTHStatus{
		WAIT_MODE: "AWAITING_TRANSACTIONS",
		RETRY_MODE:    "RETRY_TRANSACTIONS",
		PROCESSING: "PROCESSING",
		COMPLETED:  "COMPLETED",
		TERMINATED: "TERMINATED",
	}
)

// BatchRequest ... Batch request DTO for batch created for both user and system transactions
type BatchRequest struct {
	BaseModel
	AssetSymbol   string     		`json:"asset_symbol"`
	Status           string        `gorm:"index:status;not null;default:'AWAITING_TRANSACTIONS'" json:"status"`
	DateOfprocessing time.Time     `json:"date_of_processing"`
	DateCompleted    time.Time     `json:"date_completed"`
	NoOfRecords          int           `json:"no_of_records"`
	Transactions     []Transaction `json:"transaction_requests,omitempty"`
}
