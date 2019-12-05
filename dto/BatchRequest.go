package dto

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// BTHStatus ...
type BTHStatus struct {
	PENDING, PROCESSING, COMPLETED, TERMINATED, REVERSED string
}

var (
	// BatchStatus ...
	BatchStatus = BTHStatus{
		PENDING:    "Pending",
		PROCESSING: "Processing",
		COMPLETED:  "Completed",
		TERMINATED: "Terminated",
		REVERSED:   "Reversed",
	}
)

// BatchRequest ... Batch request DTO for batch created for both user and system transactions
type BatchRequest struct {
	BaseDTO
	AssetID          uuid.UUID     `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"assetId"`
	Status           string        `gorm:"index:status;not null;default:'Pending'" json:"status"`
	DateOfprocessing time.Time     `json:"dateOfprocessing"`
	DateCompleted    time.Time     `json:"dateCompleted"`
	Records          int           `json:"noOfRecords"`
	Transactions     []Transaction `json:"transactionRequests,omitempty"`
}
