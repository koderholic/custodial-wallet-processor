package model

import (
	"time"
)

// BTHStatus ...
type BTHStatus struct {
	PENDING, PROCESSING, COMPLETED, TERMINATED string
}

var (
	// BatchStatus ...
	BatchStatus = BTHStatus{
		PENDING:    "PENDING",
		PROCESSING: "PROCESSING",
		COMPLETED:  "COMPLETED",
		TERMINATED: "TERMINATED",
	}
)

// BatchRequest ... Batch request DTO for batch created for both user and system transactions
type BatchRequest struct {
	BaseModel
	AssetSymbol   string     		`json:"asset_symbol"`
	Status           string        `gorm:"index:status;not null;default:'PENDING'" json:"status"`
	DateOfprocessing time.Time     `json:"date_of_processing"`
	DateCompleted    time.Time     `json:"date_completed"`
	Records          int           `json:"no_of_records"`
	Transactions     []Transaction `json:"transaction_requests,omitempty"`
}
