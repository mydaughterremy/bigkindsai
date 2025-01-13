package model

import (
	"time"
)

type QA struct {
	ID                  string `gorm:"type:char(36);primaryKey"`
	ChatID              string `gorm:"type:char(36);index"`
	SessionID           string `gorm:"type:varchar(36);index"`
	UploadID            string `gorm:"type:char(36);index"`
	UploadDeleted       time.Time
	JobGroup            string       `gorm:"type:varchar(36);index"`
	Question            string       `gorm:"type:text;index:,class:FULLTEXT"`
	Answer              string       `gorm:"type:text"`
	References          []*Reference `gorm:"type:json;serializer:json"`
	Keywords            []string     `gorm:"type:json;serializer:json"`
	RelatedQueries      []string     `gorm:"type:json;serializer:json"`
	Vote                string       `gorm:"index"`
	CreatedAt           time.Time    `gorm:"autoCreateTime;type:datetime;index"`
	UpdatedAt           time.Time    `gorm:"autoUpdateTime;type:datetime"`
	Status              string
	TokenCount          int
	Pages               int
	EmbeddingTokenCount int
	LLMProvider         string `gorm:"index"`
	LLMModel            string `gorm:"index"`
}

type PaginationMetadata struct {
	TotalCount    int `json:"total_count"`
	ReturnedCount int `json:"returned_count"`
	CurrentOffset int `json:"current_offset"`
}

type QAsWithPagination struct {
	QAs      []*QA               `json:"qas"`
	Metadata *PaginationMetadata `json:"metadata"`
}
