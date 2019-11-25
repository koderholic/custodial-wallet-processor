package model

import (
	"time"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// BaseModel ... Shared model definition
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:VARCHAR(36);primary_key;" json:"id,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BeforeCreate will set ID field with a UUID value rather than numeric value.
func (base *BaseModel) BeforeCreate(scope *gorm.Scope) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	return scope.SetColumn("ID", uuid)
}
