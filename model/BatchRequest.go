package model

import (
	"time"
)

// BTHStatus ...
type BTHStatus struct {
	WAIT_MODE, RETRY_MODE, PROCESSING, COMPLETED, TERMINATED, START_MODE string
}

var (
	// BatchStatus ...
	BatchStatus = BTHStatus{
		WAIT_MODE:  "WAIT_MODE",
		START_MODE: "START_MODE",
		RETRY_MODE: "RETRY_MODE",
		PROCESSING: "ONGOING",
		COMPLETED:  "COMPLETED",
		TERMINATED: "TERMINATED",
	}
)

// BatchRequest ... Batch request DTO for batch created for both user and system transactions
type BatchRequest struct {
	BaseModel
	AssetSymbol      string        `json:"asset_symbol,omitempty"`
	Network      string        `json:"network,omitempty"`
	Status           string        `gorm:"index:status;not null;default:'WAIT_MODE'" json:"status"`
	DateOfProcessing *time.Time    `json:"date_of_processing,omitempty"`
	DateCompleted    *time.Time    `json:"date_completed,omitempty"`
	NoOfRecords      int           `json:"no_of_records,omitempty"`
	Transactions     []Transaction `json:"transaction_requests,omitempty"`
}
