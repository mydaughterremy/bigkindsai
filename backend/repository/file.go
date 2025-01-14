package repository

import (
	"context"

	"bigkinds.or.kr/backend/model"
	"gorm.io/gorm"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{
		db: db,
	}
}

func (f *FileRepository) CreateFile(ctx context.Context, file *model.File) error {
	res := f.db.Create(file)

	return res.Error
}
