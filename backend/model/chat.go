package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Chat struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time      `gorm:"autoCreateTime;type:datetime;index"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime;type:datetime;index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Object    string         `gorm:"-all"`
	Title     string
	SessionID string `json:"session_id" gorm:"index"`
}
