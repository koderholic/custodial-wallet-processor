package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type BTHStatus struct {
	PENDING, PROCESSING, COMPLETED, TERMINATED, REVERSED string
}

var (
	BatchStatus = BTHStatus{
		PENDING:    "Pending",
		PROCESSING: "Processing",
		COMPLETED:  "Completed",
		TERMINATED: "Terminated",
		REVERSED:   "Reversed",
	}
)

type BatchRequest struct {
	BaseModel
	cryptoId           uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"assetId"`
	ChainTransactionId uuid.UUID `gorm:"type:VARCHAR(36);" json:"chainTransactionId"`
	Status             string    `gorm:"index;not null;default:'Pending'" json:"status"`
	DateOfprocessing   time.Time `json:"dateOfprocessing"`
	DateCompleted      time.Time `json:"dateCompleted"`
	Records            int       `json:"noOfRecords"`
}
