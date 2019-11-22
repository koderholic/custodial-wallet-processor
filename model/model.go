package model

import (
	"time"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type BaseModel struct {
	ID        uuid.UUID `gorm:"type:VARCHAR(36);primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// BeforeCreate will set ID field with a UUID value rather than numeric value.
func (base *BaseModel) BeforeCreate(scope *gorm.Scope) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	return scope.SetColumn("ID", uuid)
}
