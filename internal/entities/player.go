package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Player struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string
	RoomId    uuid.UUID
}
