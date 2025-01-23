package model

import (
	"time"

	"gorm.io/gorm"
)

type Apikey struct {
	ID string `gorm:"type:char(36);primaryKey" json:"id"`
	// Apikey         string         `gorm:"type:char(36)" json:"apikey"`
	CreateAt       time.Time      `gorm:"autoCreateTime;type:datetime;index"`
	UpdatedAt      time.Time      `gorm:"autoCreateTime;type:datetime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Proposer       string         `json:"proposer"`
	Affiliation    string         `json:"affiliation"`
	Email          string         `json:"email"`
	StartDate      time.Time      `gorm:"type:datetime" json:"start_date" time_format:"2006-01-02 15:04:05"`
	EndDate        time.Time      `gorm:"type:datetime" json:"end_date" time_format:"2006-01-02 15:04:05"`
	Url            string         `json:"url"`
	Purpose        string         `json:"purpose"`
	Content        string         `json:"content"`
	Provider       string         `gorm:"type:json;serializer:json" json:"provider"`
	SummaryCount   int            `json:"summary_count"`
	TranslateCount int            `json:"translate_count"`
	ArticleCount   int            `json:"article_count"`
	FileCount      int            `json:"file_count"`
	ChatCount      int            `json:"chat_count"`
}

type ApikeyHistoryType struct {
	ID string `gorm:"type:char(36);primaryKey" json:"id"`
}

type ApikeyHistory struct {
	ID     string `gorm:"type:char(36);primaryKey" json:"id"`
	TypeID string `json:"type"`
}
