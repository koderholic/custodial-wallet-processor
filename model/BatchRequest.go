package model

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
		PENDING:    "PENDING",
		PROCESSING: "PROCESSING",
		COMPLETED:  "COMPLETED",
		TERMINATED: "TERMINATED",
		REVERSED:   "REVERSED",
	}
)

// BatchRequest ... Batch request DTO for batch created for both user and system transactions
type BatchRequest struct {
	BaseModel
	DenominationID   uuid.UUID     `gorm:"type:VARCHAR(36);not null;index:denomination_id" json:"asset_id"`
	Status           string        `gorm:"index:status;not null;default:'PENDING'" json:"status"`
	DateOfprocessing time.Time     `json:"date_of_processing"`
	DateCompleted    time.Time     `json:"date_completed"`
	Records          int           `json:"no_of_records"`
	Transactions     []Transaction `json:"transaction_requests,omitempty"`
}
