package dto

import (
	"time"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// BaseDTO ... Shared DTO definition
type BaseDTO struct {
	ID        uuid.UUID `gorm:"type:VARCHAR(36);primary_key;" json:"id,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BeforeCreate will set ID field with a UUID value rather than numeric value.
func (base *BaseDTO) BeforeCreate(scope *gorm.Scope) error {
	uuid := uuid.NewV4()
	return scope.SetColumn("ID", uuid)
}
