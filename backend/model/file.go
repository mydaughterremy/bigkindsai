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

type MultipleFileResponse struct {
	ChatId      string       `json:"chat_id"`
	UploadId    string       `json:"upload_id"`
	TotalPages  int          `json:"total_pages"`
	UploadFiles []UploadFile `json:"file_ids"`
	Filenames   []string     `json:"filenames"`
}

type UploadFile struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}
