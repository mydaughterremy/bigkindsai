package service

import (
	"context"

	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/repository"
	"github.com/google/uuid"
)

type FileService struct {
	FileRepository *repository.FileRepository
	QARepository   *repository.QARepository
}

func NewFileService() (*FileService, error) {
	return &FileService{}, nil
}

func (f *FileService) WriteUploadFiles(ctx context.Context, multipleFileResponse *model.MultipleFileResponse) error {
	for _, uploadFile := range multipleFileResponse.UploadFiles {
		file := &model.File{
			ID:       uploadFile.ID,
			Filename: uploadFile.Filename,
			UploadID: multipleFileResponse.UploadId,
		}
		err := f.FileRepository.CreateFile(ctx, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FileService) WriteUploadQA(ctx context.Context, multipleFileResponse *model.MultipleFileResponse) (*model.QA, error) {
	qa, err := f.QARepository.CreateQA(ctx, &model.QA{
		ID:        uuid.New().String(),
		ChatID:    multipleFileResponse.ChatId,
		SessionID: "",
		JobGroup:  "",
		UploadID:  multipleFileResponse.UploadId,
		Pages:     multipleFileResponse.TotalPages,
		Filenames: multipleFileResponse.Filenames,
	})

	if err != nil {
		return nil, err
	}

	return qa, nil
}
