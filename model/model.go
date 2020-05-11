package model

import (
	"time"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// BaseDTO ... Shared DTO definition
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:VARCHAR(36);primary_key;" json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate will set ID field with a UUID value rather than numeric value.
func (base *BaseModel) BeforeCreate(scope *gorm.Scope) error {
	uuid := uuid.NewV4()
	return scope.SetColumn("ID", uuid)
}
