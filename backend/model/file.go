package model

import (
	"time"

	"gorm.io/gorm"
)

type File struct {
	ID        string         `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time      `gorm:"autoCreateTime;type:datetime;index"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime;type:datetime;index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	UploadID  string         `gorm:"type:char(36)"`
	Filename  string         `gorm:"type:varchar(1000)"`
}
