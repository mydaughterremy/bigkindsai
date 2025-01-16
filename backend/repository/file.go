package repository

import (
	"context"
	"fmt"

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

func (f *FileRepository) GetFiles(ctx context.Context, uploadId string) ([]*model.File, error) {
	var files []*model.File
	if err := f.db.WithContext(ctx).Where("upload_id = ?", uploadId).Find(&files).Error; err != nil {
		return nil, err
	}

	return files, nil
}

func (f *FileRepository) CreateFile(ctx context.Context, file *model.File) error {
	res := f.db.Create(file)

	return res.Error
}

func (f *FileRepository) GetUploadId(ctx context.Context, chatId string) string {
	fmt.Println(chatId)
	var qa *model.QA
	if err := f.db.WithContext(ctx).Where("chat_id = ? AND upload_id != ''", chatId).Order("created_at desc").Limit(1).Find(&qa).Error; err != nil {
		return "error"
	}

	return qa.UploadID
}
